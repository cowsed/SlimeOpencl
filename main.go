package main

//Figure out why this stutters at a certain point
import (
	"flag"
	"fmt"
	"github.com/jgillich/go-opencl/cl"
	"image"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/inkyblackness/imgui-go"

	"github.com/cowsed/SlimeOpenCL/gui"
	//"image/png"
	"log"
	"math"
	"os"
	"runtime"
	"runtime/pprof"
	//"net/http"
	//_ "net/http/pprof"
)

const Width = 80 * 8
const Height = 60 * 8
const frames = 120000
const NumAgents = 8192 * 4

var dolog = true

var dirWindow float32 = 0.1
var dirChance float32 = 0.1

var fadeStrength float32 = 0.008
var SensorAngle float32 = math.Pi / 4.0
var diffuseStrength float32 = 0.5
var agentSpeed float32 = 1

var drawScale float32 = 1

var memprofile = flag.String("memprofile", "", "write memory profile to `file`")

func init() {
	runtime.LockOSThread()
}

func main() {
	flag.Parse()
	//Setup Logging
	log.SetOutput(new(logWriter))
	log.SetFlags(0)
	log.Println("Beginning")

	//Setup CL  ----------------------------------
	_, device, contextCl, queue := makeCLContext()
	defer contextCl.Release()

	//Load cl files
	kernelSource := loadFile("main.cl")
	log.Printf("Loaded source file. %v bytes long", len(kernelSource))
	blurSource := loadFile("gaussian_blur.cl")
	log.Printf("Loaded source file. %v bytes long", len(kernelSource))

	//Prepare kernel
	simKernel := makeKernel("simulate", kernelSource, contextCl)
	defer simKernel.Release()

	local, err := simKernel.WorkGroupSize(device)
	check(err)

	//Generate Test Data
	agentData := makeAgentData(NumAgents, Width, Height)

	//Create Buffers
	agentBuffer, err := contextCl.CreateEmptyBuffer(cl.MemReadWrite, 4*len(agentData))
	defer agentBuffer.Release()
	check(err)

	Image1 := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{Width, Height}})
	imgBuf1, err := contextCl.CreateImageSimple(cl.MemReadWrite|cl.MemUseHostPtr, Width, Height, cl.ChannelOrderRGBA, cl.ChannelDataTypeUNormInt8, Image1.Pix)
	defer imgBuf1.Release()
	check(err)

	Image2 := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{Width, Height}})
	imgBuf2, err := contextCl.CreateImageSimple(cl.MemReadWrite|cl.MemUseHostPtr, Width, Height, cl.ChannelOrderRGBA, cl.ChannelDataTypeUNormInt8, Image2.Pix)
	defer imgBuf2.Release()
	check(err)

	//Write Agent Data
	queue.EnqueueWriteBufferFloat32(agentBuffer, true, 0, agentData[:], nil)
	check(err)

	//Adjust Global size to make it work
	global := NumAgents
	d := len(agentData) / 3 % local
	if d != 0 {
		global += local - d
	}

	//Blur kernel
	blurKernel := makeKernel("gaussian_blur", blurSource, contextCl)
	defer blurKernel.Release()

	globalBlur := Width * Height
	d = Width * Height % local
	if d != 0 {
		globalBlur += local - d
	}

	//Setup GL
	platform, contextGL, imguiRenderer, imguiInput, r := setupGL()
	//IO:=imgui.CurrentIO()

	defer contextGL.Destroy()
	defer imguiRenderer.Dispose()

	frameNum := 0
	tex, err := createImageTexture(Image1)
	check(err)

	for !platform.ShouldStop() {
		//Do Window Stuff
		platform.ProcessEvents()

		cursorX, cursorY := platform.GetCursorPos()
		mouseState := gui.ImguiMouseState{
			MousePosX:  float32(cursorX),
			MousePosY:  float32(cursorY),
			MousePress: platform.GetMousePresses123()}
		imguiInput.NewFrame(platform.DisplaySize()[0], platform.DisplaySize()[1], glfw.GetTime(), platform.IsFocused(), mouseState)

		//Execute Simulator
		simKernel.SetArgs(agentBuffer, uint32(Width), uint32(Height), dirChance, dirWindow, SensorAngle, agentSpeed, imgBuf1, imgBuf2, uint32(NumAgents), uint32(frameNum))
		check(err)
		e, err := queue.EnqueueNDRangeKernel(simKernel, nil, []int{global}, []int{local}, nil)
		check(err)
		err = queue.Finish()
		check(err)
		e.Release()

		//Swap Buffers
		tempBuf := imgBuf1
		imgBuf1 = imgBuf2
		imgBuf2 = tempBuf
		//Execute Blur
		err = blurKernel.SetArgs(imgBuf1, imgBuf2, fadeStrength, diffuseStrength, uint32(Width), uint32(Height))
		e, err = queue.EnqueueNDRangeKernel(blurKernel, nil, []int{globalBlur}, []int{local}, nil)
		check(err)
		err = queue.Finish()
		check(err)
		e.Release()
		/*
			//Swap Buffers
			tempBuf = imgBuf1
			imgBuf1 = imgBuf2
			imgBuf2 = tempBuf
		*/

		//Read image
		e, err = queue.EnqueueReadImage(imgBuf2, true, [3]int{0, 0, 0}, [3]int{Width, Height, 1}, Width*4, 0, Image1.Pix, nil)
		check(err)
		e.Release()
		//fblur, _ := os.Create(fmt.Sprintf("Output/blur%d.png", frameNum))
		//png.Encode(fblur, Image1)


		//HERE COMES THE MEMORY LEAK
		gl.DeleteTextures(1, &(tex))
		tex, err := createImageTexture(Image1)
		r.texture = tex

		check(err)
		//Imgui stuff
		imgui.Begin("Parameter Editor")
		

		imgui.Text(fmt.Sprintf("Frame: %d", frameNum))
		//imgui.Image(imgui.TextureID(tex),imgui.Vec2{Width/2,Height/2})
		imgui.SliderFloat("Fade Strength", &fadeStrength, 0, 1.0)
		imgui.SliderFloat("Diffuse Strength", &diffuseStrength, 0, 1.0)

		imgui.SliderFloat("Sensor Angle", &SensorAngle, 0, 1.0)
		imgui.SliderFloat("Agent Speed", &agentSpeed, 0, 18)

		imgui.SliderFloat("Turn Window", &dirWindow, 0, 1)

		
		if imgui.Button("Log") {
			if *memprofile != "" || dolog {
				fmt.Println("Write to profile")

				f, err := os.Create("mem.pprof")
				if err != nil {
					log.Fatal("could not create memory profile: ", err)
				}
				defer f.Close() // error handling omitted for example
				runtime.GC()    // get up-to-date statistics
				//profiler.WriteTo(f, 0)
				if err := pprof.WriteHeapProfile(f); err != nil {
					log.Fatal("could not write memory profile: ", err)
				}
			}
		}
		imgui.End()
		imgui.Render()

		//Clean up after imgui
		cleanUpAfterImgui()
		r.Draw(drawScale)

		imguiRenderer.Render(platform.DisplaySize(), platform.FramebufferSize(), imgui.RenderedDrawData())

		platform.PostRender()
		frameNum++

	}

}
