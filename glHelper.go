package main

import (
	"fmt"
	"github.com/inkyblackness/imgui-go"
	"os"

	"github.com/go-gl/gl/v3.2-core/gl"
	"github.com/cowsed/SlimeOpenCL/gui"
	"github.com/cowsed/SlimeOpenCL/platform"
)

var fragmentShaderSource = `#version 410
in vec2 UV;

out vec4 frag_colour;
void main() {
	frag_colour = vec4(UV.x,UV.y, 0, 1);
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
func cleanUpAfterImgui(){
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.Enable(gl.DEPTH_TEST)
		gl.DepthFunc(gl.LESS)
		gl.ClearColor(1, 0, 0, 1)
}

func setupGL() (*platform.Platform, *imgui.Context, *gui.OpenGL3, *gui.ImguiInput) {
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

	return platform, context, imguiRenderer, &imguiInput
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
