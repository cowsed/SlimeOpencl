package main

import (
	"fmt"
	"github.com/cowsed/SlimeOpenCL/gui"
	"github.com/cowsed/SlimeOpenCL/platform"
	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/inkyblackness/imgui-go"
	"image"
	"os"
)

//Wam Bam memory leak
//Still dont know why this leaks and doesnt release correclyl when gl.DeleteTextures is called but found a way around it
//Investigate the siurce of this whihc i think is the imgui-go package
func createImageTexture(img *image.RGBA) (uint32, error) {
	// Upload texture to graphics system
	var lastTexture int32
	var handle uint32
	gl.GetIntegerv(gl.TEXTURE_BINDING_2D, &lastTexture)
	gl.GenTextures(1, &handle)
	gl.BindTexture(gl.TEXTURE_2D, handle)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR) // minification filter
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR) // magnification filter
	//This line specifically v
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA, int32(img.Bounds().Dx()), int32(img.Bounds().Dy()), 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(&img.Pix[0]))
	//gl.GenerateMipmap(gl.TEXTURE_2D)

	// Restore state
	gl.BindTexture(gl.TEXTURE_2D, uint32(lastTexture))

	return handle, nil
}

var fragmentShaderSource = `#version 410
in vec2 UV;
uniform sampler2D image;

out vec4 frag_colour;
void main() {
	frag_colour = texture2D(image,UV*0.5-0.5);//+vec4(UV.x,UV.y, 0, 1);
}
`
var vertexShaderSource = `#version 410
in vec3 vp;
in vec2 uv_in;
out vec2 UV;

void main() {
	UV=vp.yx;
	gl_Position = vec4(vp, 1.0);
}
` + "\x00"

//Sets gl parameters to work with wacky imgui stuff
func cleanUpAfterImgui() {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.ClearColor(1, 0, 0, 1)
}

func setupGL() (*platform.Platform, *imgui.Context, *gui.OpenGL3, *gui.ImguiInput, *Renderer) {
	//Setup IMGUI
	context, imguiInput := gui.NewImgui()

	// Setup the GLFW platform
	platform, err := platform.NewPlatform(Width, Height)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}

	clipboard := CombinedClipboard{platform}
	IO := imgui.CurrentIO()
	IO.SetClipboard(clipboard)

	// Setup the Imgui renderer
	imguiRenderer, err := gui.NewOpenGL3()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(-1)
	}

	// Setup the platform callbacks
	platform.SetMouseButtonCallback(imguiInput.MouseButtonChange)
	platform.SetScrollCallback(imguiInput.MouseScrollChange)
	platform.SetKeyCallback(imguiInput.KeyChange)
	platform.SetCharCallback(imguiInput.CharChange)

	version := gl.GoStr(gl.GetString(gl.VERSION))
	fmt.Println("OpenGL version", version)

	//Set up the renderer
	r := new(Renderer)

	r.Init()
	r.UpdateProgram(fragmentShaderSource, vertexShaderSource)

	return platform, context, imguiRenderer, &imguiInput, r
}

//Clipboard
type CombinedClipboard struct {
	platform *platform.Platform
}

func (c CombinedClipboard) Text() (string, error) {
	return c.platform.ClipboardText()
}
func (c CombinedClipboard) SetText(value string) {
	c.platform.SetClipboardText(value)
}
