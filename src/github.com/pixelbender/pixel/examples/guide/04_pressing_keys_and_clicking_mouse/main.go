package main

import (
	"image"
	"math"
	"math/rand"
	"os"
	"time"

	_ "image/png"

	"github.com/faiface/pixel"
	"github.com/faiface/pixel/pixelgl"
	"golang.org/x/image/colornames"
)

func loadPicture(path string) (pixel.Picture, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}
	return pixel.PictureDataFromImage(img), nil
}

func run() {
	cfg := pixelgl.WindowConfig{
		Title:  "Pixel Rocks!",
		Bounds: pixel.R(0, 0, 1024, 768),
		VSync:  true,
	}
	win, err := pixelgl.NewWindow(cfg)
	if err != nil {
		panic(err)
	}

	spritesheet, err := loadPicture("trees.png")
	if err != nil {
		panic(err)
	}

	var treesFrames []pixel.Rect
	for x := spritesheet.Bounds().Min.X(); x < spritesheet.Bounds().Max.X(); x += 32 {
		for y := spritesheet.Bounds().Min.Y(); y < spritesheet.Bounds().Max.Y(); y += 32 {
			treesFrames = append(treesFrames, pixel.R(x, y, x+32, y+32))
		}
	}

	var (
		camPos       = pixel.V(0, 0)
		camSpeed     = 500.0
		camZoom      = 1.0
		camZoomSpeed = 1.2
		trees        []*pixel.Sprite
	)

	last := time.Now()
	for !win.Closed() {
		dt := time.Since(last).Seconds()
		last = time.Now()

		cam := pixel.IM.Scaled(camPos, camZoom).Moved(win.Bounds().Center() - camPos)
		win.SetMatrix(cam)

		if win.JustPressed(pixelgl.MouseButtonLeft) {
			tree := pixel.NewSprite(spritesheet, treesFrames[rand.Intn(len(treesFrames))])
			mouse := cam.Unproject(win.MousePosition())
			tree.SetMatrix(pixel.IM.Scaled(0, 4).Moved(mouse))
			trees = append(trees, tree)
		}
		if win.Pressed(pixelgl.KeyLeft) {
			camPos -= pixel.X(camSpeed * dt)
		}
		if win.Pressed(pixelgl.KeyRight) {
			camPos += pixel.X(camSpeed * dt)
		}
		if win.Pressed(pixelgl.KeyDown) {
			camPos -= pixel.Y(camSpeed * dt)
		}
		if win.Pressed(pixelgl.KeyUp) {
			camPos += pixel.Y(camSpeed * dt)
		}
		camZoom *= math.Pow(camZoomSpeed, win.MouseScroll().Y())

		win.Clear(colornames.Forestgreen)

		for _, tree := range trees {
			tree.Draw(win)
		}

		win.Update()
	}
}

func main() {
	pixelgl.Run(run)
}
