package main

import (
	"github.com/jgillich/go-opencl/cl"
	"io/ioutil"
	"log"
	"math"
)

func makeAgentData(n, Width, Height int) []float32 {
	//Positionx/ posy, direction
	agentData := make([]float32, 3*n)
	for i := 0; i < n*3; i += 3 {
		a := float32(i) / float32(n) * 2 * math.Pi
		r := 100.0
		agentData[i] = float32(Width/2) + float32(math.Cos(float64(a))*r)
		agentData[i+1] = float32(Height/2) + float32(math.Sin(float64(a))*r)
		agentData[i+2] = a

	}
	return agentData
}

func createBlurMask(sigma float64) []float32 {
	maskSize := int(math.Ceil(3.0 * sigma))
	mask := make([]float32, (maskSize*2+1)*(maskSize*2+1))
	var sum float32 = 0.0
	for a := -maskSize; a < maskSize+1; a++ {
		for b := -maskSize; b < maskSize+1; b++ {
			var temp float32 = float32(math.Exp(-(float64(a*a+b*b) / (2 * sigma * sigma))))
			sum += temp
			mask[a+maskSize+(b+maskSize)*(maskSize*2+1)] = temp
		}
	}
	// Normalize the mask
	for i := 0; i < (maskSize*2+1)*(maskSize*2+1); i++ {
		mask[i] = mask[i] / float32(sum)
	}

	return mask
}

func makeKernel(kernelName string, kernelSource string, context *cl.Context) *cl.Kernel {
	log.Println("Creating Kernel", kernelName)
	program, err := context.CreateProgramWithSource([]string{kernelSource})
	check(err)
	err = program.BuildProgram(nil, "")
	check(err)
	kernel, err := program.CreateKernel(kernelName)
	check(err)

	numArgs, err := kernel.NumArgs()
	check(err)
	log.Printf("%v arguements detected", numArgs)

	for i := 0; i < numArgs; i++ {
		name, err := kernel.ArgName(i)
		if err == cl.ErrUnsupported {
			break
		} else if err != nil {
			log.Printf("GetKernelArgInfo for name:%v failed: %+v", name, err)
			break
		} else {
			log.Printf("Kernel arg %d: %s \n", i, name)
		}
	}

	return kernel
}

func makeCLContext() (*cl.Platform, *cl.Device, *cl.Context, *cl.CommandQueue) {
	//Load Platform
	platforms, err := cl.GetPlatforms()
	check(err)
	platform := platforms[0]
	log.Printf("Loaded Platform: %v\n", platform.Name())

	//Setup opencl
	devices, err := platform.GetDevices(cl.DeviceTypeGPU)
	check(err)
	device := devices[0]
	log.Printf("Loaded Device: %v", device.Name())

	context, err := cl.CreateContext([]*cl.Device{device})
	check(err)

	queue, err := context.CreateCommandQueue(device, 0)
	check(err)
	return platform, device, context, queue
}

func loadFile(fname string) string {
	content, err := ioutil.ReadFile(fname)
	check(err)
	return string(content)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
