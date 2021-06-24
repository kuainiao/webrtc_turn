package pixelgl

import (
	"image/color"
	"runtime"

	"github.com/faiface/glhf"
	"github.com/faiface/mainthread"
	"github.com/faiface/pixel"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/pkg/errors"
)

// WindowConfig is a structure for specifying all possible properties of a Window. Properties are
// chosen in such a way, that you usually only need to set a few of them - defaults (zeros) should
// usually be sensible.
//
// Note that you always need to set the Bounds of a Window.
type WindowConfig struct {
	// Title at the top of the Window.
	Title string

	// Bounds specify the bounds of the Window in pixels.
	Bounds pixel.Rect

	// If set to nil, the Window will be windowed. Otherwise it will be fullscreen on the
	// specified Monitor.
	Monitor *Monitor

	// Whether the Window is resizable.
	Resizable bool

	// Undecorated Window ommits the borders and decorations (close button, etc.).
	Undecorated bool

	// VSync (vertical synchronization) synchronizes Window's framerate with the framerate of
	// the monitor.
	VSync bool
}

// Window is a window handler. Use this type to manipulate a window (input, drawing, etc.).
type Window struct {
	window *glfw.Window

	bounds pixel.Rect
	canvas *Canvas
	vsync  bool

	// need to save these to correctly restore a fullscreen window
	restore struct {
		xpos, ypos, width, height int
	}

	prevInp, tempInp, currInp struct {
		mouse   pixel.Vec
		buttons [KeyLast + 1]bool
		scroll  pixel.Vec
	}
}

var currWin *Window

// NewWindow creates a new Window with it's properties specified in the provided config.
//
// If Window creation fails, an error is returned (e.g. due to unavailable graphics device).
func NewWindow(cfg WindowConfig) (*Window, error) {
	bool2int := map[bool]int{
		true:  glfw.True,
		false: glfw.False,
	}

	w := &Window{bounds: cfg.Bounds}

	err := mainthread.CallErr(func() error {
		var err error

		glfw.WindowHint(glfw.ContextVersionMajor, 3)
		glfw.WindowHint(glfw.ContextVersionMinor, 3)
		glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
		glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

		glfw.WindowHint(glfw.Resizable, bool2int[cfg.Resizable])
		glfw.WindowHint(glfw.Decorated, bool2int[!cfg.Undecorated])

		var share *glfw.Window
		if currWin != nil {
			share = currWin.window
		}
		_, _, width, height := intBounds(cfg.Bounds)
		w.window, err = glfw.CreateWindow(
			width,
			height,
			cfg.Title,
			nil,
			share,
		)
		if err != nil {
			return err
		}

		// enter the OpenGL context
		w.begin()
		w.end()

		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating window failed")
	}

	w.SetVSync(cfg.VSync)

	w.initInput()
	w.SetMonitor(cfg.Monitor)

	w.canvas = NewCanvas(cfg.Bounds)
	w.Update()

	runtime.SetFinalizer(w, (*Window).Destroy)

	return w, nil
}

// Destroy destroys the Window. The Window can't be used any further.
func (w *Window) Destroy() {
	mainthread.Call(func() {
		w.window.Destroy()
	})
}

// Update swaps buffers and polls events. Call this method at the end of each frame.
func (w *Window) Update() {
	mainthread.Call(func() {
		_, _, oldW, oldH := intBounds(w.bounds)
		newW, newH := w.window.GetSize()
		w.bounds = w.bounds.ResizedMin(w.bounds.Size() + pixel.V(
			float64(newW-oldW),
			float64(newH-oldH),
		))
	})

	w.canvas.SetBounds(w.bounds)

	mainthread.Call(func() {
		w.begin()

		framebufferWidth, framebufferHeight := w.window.GetFramebufferSize()
		glhf.Bounds(0, 0, framebufferWidth, framebufferHeight)

		glhf.Clear(0, 0, 0, 0)
		w.canvas.gf.Frame().Begin()
		w.canvas.gf.Frame().Blit(
			nil,
			0, 0, w.canvas.Texture().Width(), w.canvas.Texture().Height(),
			0, 0, framebufferWidth, framebufferHeight,
		)
		w.canvas.gf.Frame().End()

		if w.vsync {
			glfw.SwapInterval(1)
		} else {
			glfw.SwapInterval(0)
		}
		w.window.SwapBuffers()
		w.end()
	})

	w.updateInput()
}

// SetClosed sets the closed flag of the Window.
//
// This is useful when overriding the user's attempt to close the Window, or just to close the
// Window from within the program.
func (w *Window) SetClosed(closed bool) {
	mainthread.Call(func() {
		w.window.SetShouldClose(closed)
	})
}

// Closed returns the closed flag of the Window, which reports whether the Window should be closed.
//
// The closed flag is automatically set when a user attempts to close the Window.
func (w *Window) Closed() bool {
	var closed bool
	mainthread.Call(func() {
		closed = w.window.ShouldClose()
	})
	return closed
}

// SetTitle changes the title of the Window.
func (w *Window) SetTitle(title string) {
	mainthread.Call(func() {
		w.window.SetTitle(title)
	})
}

// SetBounds sets the bounds of the Window in pixels. Bounds can be fractional, but the actual size
// of the window will be rounded to integers.
func (w *Window) SetBounds(bounds pixel.Rect) {
	w.bounds = bounds
	mainthread.Call(func() {
		_, _, width, height := intBounds(bounds)
		w.window.SetSize(width, height)
	})
}

// Bounds returns the current bounds of the Window.
func (w *Window) Bounds() pixel.Rect {
	return w.bounds
}

func (w *Window) setFullscreen(monitor *Monitor) {
	mainthread.Call(func() {
		w.restore.xpos, w.restore.ypos = w.window.GetPos()
		w.restore.width, w.restore.height = w.window.GetSize()

		mode := monitor.monitor.GetVideoMode()

		w.window.SetMonitor(
			monitor.monitor,
			0,
			0,
			mode.Width,
			mode.Height,
			mode.RefreshRate,
		)
	})
}

func (w *Window) setWindowed() {
	mainthread.Call(func() {
		w.window.SetMonitor(
			nil,
			w.restore.xpos,
			w.restore.ypos,
			w.restore.width,
			w.restore.height,
			0,
		)
	})
}

// SetMonitor sets the Window fullscreen on the given Monitor. If the Monitor is nil, the Window
// will be restored to windowed state instead.
//
// The Window will be automatically set to the Monitor's resolution. If you want a different
// resolution, you will need to set it manually with SetBounds method.
func (w *Window) SetMonitor(monitor *Monitor) {
	if w.Monitor() != monitor {
		if monitor != nil {
			w.setFullscreen(monitor)
		} else {
			w.setWindowed()
		}
	}
}

// Monitor returns a monitor the Window is fullscreen on. If the Window is not fullscreen, this
// function returns nil.
func (w *Window) Monitor() *Monitor {
	var monitor *glfw.Monitor
	mainthread.Call(func() {
		monitor = w.window.GetMonitor()
	})
	if monitor == nil {
		return nil
	}
	return &Monitor{
		monitor: monitor,
	}
}

// Focused returns true if the Window has input focus.
func (w *Window) Focused() bool {
	var focused bool
	mainthread.Call(func() {
		focused = w.window.GetAttrib(glfw.Focused) == glfw.True
	})
	return focused
}

// SetVSync sets whether the Window's Update should synchronize with the monitor refresh rate.
func (w *Window) SetVSync(vsync bool) {
	w.vsync = vsync
}

// VSync returns whether the Window is set to synchronize with the monitor refresh rate.
func (w *Window) VSync() bool {
	return w.vsync
}

// Note: must be called inside the main thread.
func (w *Window) begin() {
	if currWin != w {
		w.window.MakeContextCurrent()
		glhf.Init()
		currWin = w
	}
}

// Note: must be called inside the main thread.
func (w *Window) end() {
	// nothing, really
}

// MakeTriangles generates a specialized copy of the supplied Triangles that will draw onto this
// Window.
//
// Window supports TrianglesPosition, TrianglesColor and TrianglesPicture.
func (w *Window) MakeTriangles(t pixel.Triangles) pixel.TargetTriangles {
	return w.canvas.MakeTriangles(t)
}

// MakePicture generates a specialized copy of the supplied Picture that will draw onto this Window.
//
// Window supports PictureColor.
func (w *Window) MakePicture(p pixel.Picture) pixel.TargetPicture {
	return w.canvas.MakePicture(p)
}

// SetMatrix sets a Matrix that every point will be projected by.
func (w *Window) SetMatrix(m pixel.Matrix) {
	w.canvas.SetMatrix(m)
}

// SetColorMask sets a global color mask for the Window.
func (w *Window) SetColorMask(c color.Color) {
	w.canvas.SetColorMask(c)
}

// SetComposeMethod sets a Porter-Duff composition method to be used in the following draws onto
// this Window.
func (w *Window) SetComposeMethod(cmp pixel.ComposeMethod) {
	w.canvas.SetComposeMethod(cmp)
}

// SetSmooth sets whether the stretched Pictures drawn onto this Window should be drawn smooth or
// pixely.
func (w *Window) SetSmooth(smooth bool) {
	w.canvas.SetSmooth(smooth)
}

// Smooth returns whether the stretched Pictures drawn onto this Window are set to be drawn smooth
// or pixely.
func (w *Window) Smooth() bool {
	return w.canvas.Smooth()
}

// Clear clears the Window with a single color.
func (w *Window) Clear(c color.Color) {
	w.canvas.Clear(c)
}

// Color returns the color of the pixel over the given position inside the Window.
func (w *Window) Color(at pixel.Vec) pixel.RGBA {
	return w.canvas.Color(at)
}
