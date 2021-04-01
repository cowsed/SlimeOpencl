package main

import (
	"fmt"
	"github.com/jgillich/go-opencl/cl"
	"image"

	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/inkyblackness/imgui-go"

	"github.com/cowsed/OpenCLSlime/gui"
	"image/png"
	"log"
	"os"
	"runtime"
)

const Width = 800
const Height = 600
const frames = 120
const NumAgents = 8192
const perFrame = 2
const dirWindow float32 = 0.1
const dirChance float32 = 0.1

func init() {
	runtime.LockOSThread()
}

func main() {

	//Setup Logging
	log.SetOutput(new(logWriter))
	log.SetFlags(0)

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
	platform, contextGL, imguiRenderer, imguiInput:= setupGL()
	IO:=imgui.CurrentIO()

	defer contextGL.Destroy()
	defer imguiRenderer.Dispose()

	frameNum := 0
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
		simKernel.SetArgs(agentBuffer, uint32(Width), uint32(Height), dirChance, dirWindow, imgBuf1, imgBuf2, uint32(NumAgents), uint32(frameNum))
		check(err)
		_, err = queue.EnqueueNDRangeKernel(simKernel, nil, []int{global}, []int{local}, nil)
		check(err)
		err = queue.Finish()
		check(err)

		//Swap Buffers
		tempBuf := imgBuf1
		imgBuf1 = imgBuf2
		imgBuf2 = tempBuf
		//Execute Blur
		err = blurKernel.SetArgs(imgBuf1, imgBuf2, uint32(Width), uint32(Height))
		_, err = queue.EnqueueNDRangeKernel(blurKernel, nil, []int{globalBlur}, []int{local}, nil)
		check(err)
		err = queue.Finish()
		check(err)

		//Swap Buffers
		//tempBuf := imgBuf1
		//imgBuf1 = imgBuf2
		//imgBuf2 = tempBuf

		//Read image
		_, err = queue.EnqueueReadImage(imgBuf2, true, [3]int{0, 0, 0}, [3]int{Width, Height, 1}, Width*4, 0, Image1.Pix, nil)
		fblur, _ := os.Create(fmt.Sprintf("Output/blur%d.png", frameNum))
		png.Encode(fblur, Image1)
		frameNum++

		//Imgui stuff
		imgui.Begin("Parameter Editor")
		imgui.Text(fmt.Sprintf("Frame: %d", frameNum))
		imgui.Text(fmt.Sprintf("FPS: %v", IO.Framerate()))
		
		imgui.End()
		imgui.Render()
		//Clean up after imgui
		cleanUpAfterImgui()
		
		imguiRenderer.Render(platform.DisplaySize(), platform.FramebufferSize(), imgui.RenderedDrawData())
		platform.PostRender()

		if frameNum == frames {
			break
		}
	}

}
