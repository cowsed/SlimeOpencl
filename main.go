package main

import (
	"fmt"
	"github.com/jgillich/go-opencl/cl"
	"image"

	"image/png"
	"log"
	"os"
	"time"
)

const Width = 800  //4
const Height = 600 //4
const NumAgents = 2048
const frames = 60


func main() {

	//Setup Logging
	log.SetOutput(new(logWriter))
	log.SetFlags(0)

	//Setup CL
	_, device, context, queue := makeCLContext()

	defer context.Release()

	//Load cl file
	kernelSource := loadFile("main.cl")
	log.Printf("Loaded source file. %v bytes long", len(kernelSource))
	blurSource := loadFile("gaussian_blur.cl")
	log.Printf("Loaded source file. %v bytes long", len(kernelSource))

	//Prepare kernel
	simKernel := makeKernel("simulate", kernelSource, context)
	defer simKernel.Release()

	local, err := simKernel.WorkGroupSize(device)
	check(err)

	//Generate Test Data
	agentData := makeAgentData(NumAgents, Width, Height)

	//Create Buffers
	agentBuffer, err := context.CreateEmptyBuffer(cl.MemReadWrite, 4*len(agentData))
	defer agentBuffer.Release()
	check(err)


	Image1 := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{Width, Height}})
	imgBuf1, err := context.CreateImageSimple(cl.MemReadWrite|cl.MemUseHostPtr, Width, Height, cl.ChannelOrderRGBA, cl.ChannelDataTypeUNormInt8, Image1.Pix)
	defer imgBuf1.Release()
	check(err)

	Image2 := image.NewRGBA(image.Rectangle{image.Point{0, 0}, image.Point{Width, Height}})
	imgBuf2, err := context.CreateImageSimple(cl.MemReadWrite|cl.MemUseHostPtr, Width, Height, cl.ChannelOrderRGBA, cl.ChannelDataTypeUNormInt8, Image2.Pix)
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

	check(err)


	//Blur
	blurKernel := makeKernel("gaussian_blur", blurSource, context)
	defer blurKernel.Release()




	check(err)
	moreFull := time.Now()
	for i := 0; i < frames; i++ {
		log.Printf("Frame %d", i)
		//Execute Simulator
		simKernel.SetArgs(agentBuffer, uint32(Width), uint32(Height), imgBuf1, imgBuf2, uint32(NumAgents))
		check(err)
		_, err = queue.EnqueueNDRangeKernel(simKernel, nil, []int{global}, []int{local}, nil)
		check(err)

		err = queue.Finish()
		check(err)

		//Execute Blur
		err = blurKernel.SetArgs(imgBuf2, imgBuf1, uint32(Width), uint32(Height))

		_, err = queue.EnqueueNDRangeKernel(blurKernel, nil, []int{Width * Height}, []int{local}, nil)
		check(err)

		err = queue.Finish()
		check(err)

		//Swap Buffers
		tempBuf := imgBuf1
		imgBuf1 = imgBuf2
		imgBuf2 = tempBuf

		check(err)

		//Save to file
		//This may be weird cuz at one point i had to read from the image but now i dont think i do

		//Read from buffer
		_, err = queue.EnqueueReadImage(imgBuf1, true, [3]int{0, 0, 0}, [3]int{Width, Height, 1}, Width*4, 0, Image1.Pix, nil)

		fblur, _ := os.Create(fmt.Sprintf("Output/blur%d.png", i))
		png.Encode(fblur, Image1)
	}
	fullElapsed := time.Since(moreFull)
	log.Println("Full loop took", fullElapsed)


	fmt.Println("Before:", agentData[:10])
	_, err = queue.EnqueueReadBufferFloat32(agentBuffer, true, 0, agentData, nil)
	fmt.Println("After:", agentData[:10])

}

//Custom Log Writer
type logWriter struct {
}

func (writer logWriter) Write(bytes []byte) (int, error) {
	return fmt.Print(" [DEBUG] " + string(bytes))
}
