package main

import (
	"C"
	"fmt"
	"github.com/go-gl/gl/v3.2-core/gl"
	"strings"
	"unsafe"
)

type Renderer struct {
	vao         uint32
	vbo         uint32
	points      []float32
	program     uint32
	errorString string
}

func (r *Renderer) Init() {
	r.points = []float32{
		-1, 1, 0,
		-1, -1, 0,
		1, -1, 0,

		-1, 1, 0,
		1, 1, 0,
		1, -1, 0,
	}
	//Make the vao
	r.vao = makeVao(r.points)
}
func (r *Renderer) UpdateProgram(fragSource, vertSource string) {
	fmt.Println("Updating Program")
	r.errorString = ""
	vertexShader, err := compileShader(vertSource, gl.VERTEX_SHADER)
	if err != nil {
		//Only panic here because this should never happen
		panic(err)
	}

	fragmentShader, err := compileShader(fragSource, gl.FRAGMENT_SHADER)
	if err != nil {
		r.errorString += err.Error()
		return
	}
	//Check Fragment Shader Error

	prog := gl.CreateProgram()
	gl.AttachShader(prog, vertexShader)
	gl.AttachShader(prog, fragmentShader)
	gl.LinkProgram(prog)

	var isLinked int32
	gl.GetProgramiv(prog, gl.LINK_STATUS, &isLinked)
	fmt.Println("Program Link Error ", isLinked)
	if isLinked == gl.FALSE {
		var maxLength int32
		gl.GetProgramiv(fragmentShader, gl.INFO_LOG_LENGTH, &maxLength)

		infoLog := make([]uint8, maxLength+1) //[bufSize]uint8{}
		gl.GetShaderInfoLog(fragmentShader, maxLength, &maxLength, &infoLog[0])

		r.errorString += fmt.Sprintln("Link Infolog{", string(infoLog), "}")
		return
	}

	r.program = prog
	r.errorString += "Shader Compiled Succesfully"
}
func (r *Renderer) Draw() {
	gl.UseProgram(r.program)
	gl.BindVertexArray(r.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(r.points)/3))
}
func (r *Renderer) DrawToFramebuffer() uint32 {
	//Generate new framebuffer
	var frameBuffer uint32
	gl.GenFramebuffers(1, &frameBuffer)

	gl.BindFramebuffer(gl.FRAMEBUFFER, frameBuffer)

	gl.UseProgram(r.program)
	gl.BindVertexArray(r.vao)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(len(r.points)/3))

	//Switch back to default framebuffer
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

	fmt.Println("Made fbo", frameBuffer)
	return frameBuffer
}

func FrameBufferToFile(filename string, framebuffer uint32, width, height int) {
	fmt.Println("Saving framebuffer", framebuffer, " to file")
	//Make Sure framebuffer is finished before drawing

	status := gl.CheckNamedFramebufferStatus(gl.DRAW_FRAMEBUFFER, framebuffer)
	fmt.Println("Status: ", status)

	var pixels unsafe.Pointer
	format := uint32(gl.RGBA)
	xtype := uint32(gl.UNSIGNED_INT_8_8_8_8)
	//Activate framebuffer
	gl.BindFramebuffer(gl.FRAMEBUFFER, framebuffer)

	gl.PixelStorei(gl.PACK_ALIGNMENT, 1);
	//Read pixels
	gl.ReadPixels(0, 0, int32(width), int32(height), format, xtype, pixels)
	//size=int32(width*height*4)
	print("Pixel data: ")
	fmt.Println(pixels)



	//Set back to defualt frame buffer
	gl.BindFramebuffer(gl.FRAMEBUFFER, 0)

}

//makes vao
func makeVao(points []float32) uint32 {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)

	return vao
}

//Compiles shaders
func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile:\n %v\nLog:\n%v", source[:len(source)-1], log[:len(log)-1])
	}

	return shader, nil
}
