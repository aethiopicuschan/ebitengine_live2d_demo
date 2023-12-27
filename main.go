package main

import (
	_ "embed"
	"fmt"
	"image/color"
	_ "image/png"
	"log"
	"path/filepath"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/colorm"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed  shader.kage
var shaderSrc []byte

const (
	DEFAULT_WIDTH  = 640
	DEFAULT_HEIGHT = 480
	NAME           = "Haru"
)

var BlendForDraw = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorOne,
	BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
	BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceAlpha,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

var BlendForMask = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorOne,
	BlendFactorSourceAlpha:      ebiten.BlendFactorOne,
	BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceAlpha,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

var BlendForMask2 = ebiten.Blend{
	BlendFactorSourceRGB:        ebiten.BlendFactorZero,
	BlendFactorSourceAlpha:      ebiten.BlendFactorZero,
	BlendFactorDestinationRGB:   ebiten.BlendFactorOneMinusSourceColor,
	BlendFactorDestinationAlpha: ebiten.BlendFactorOneMinusSourceAlpha,
	BlendOperationRGB:           ebiten.BlendOperationAdd,
	BlendOperationAlpha:         ebiten.BlendOperationAdd,
}

type Drawable struct {
	textureIndex             int
	vertexPositions          []Vertex2
	vertexUVs                []Vertex2
	vertexIndices            []uint16
	id                       uintptr
	opacity                  float32
	blendMode                int // 通常:0 加算:1 乗算:2 マスク:3
	masks                    []int32
	isInvertedMask           bool
	culling                  int
	isVisible                bool
	vertexPositionsDidChange bool
	vertices                 []ebiten.Vertex
}

type Vertex2 struct {
	X float32
	Y float32
}

type Game struct {
	cubism                                         uintptr
	shader                                         *ebiten.Shader
	textureMap                                     map[int]*ebiten.Image
	parameterMap                                   map[string]int
	drawables                                      []Drawable
	renderOrders                                   []int
	surface                                        *ebiten.Image
	maskBuffer                                     *ebiten.Image
	frameBuffer                                    *ebiten.Image
	scaleX                                         float64
	scaleY                                         float64
	width, height                                  int
	newCubism                                      func(int) uintptr
	loadModel                                      func(uintptr, string, string)
	getDrawableCount                               func(uintptr) int
	getTextureIndex                                func(uintptr, int) int
	getDrawableVertexCount                         func(uintptr, int) int
	getDrawableVertexPositions                     func(uintptr, int) uintptr
	getDrawableVertexUVs                           func(uintptr, int) uintptr
	getIndexCount                                  func(uintptr, int) int
	getDrawableVertexIndices                       func(uintptr, int) uintptr
	getDrawableId                                  func(uintptr, int) uintptr
	getDrawableOpacity                             func(uintptr, int) float32
	getDrawableRenderOrders                        func(uintptr) uintptr
	getDrawableBlendMode                           func(uintptr, int) int
	getMaskCount                                   func(uintptr, int) int
	getMasks                                       func(uintptr, int) uintptr
	getDrawableIsInvertedMask                      func(uintptr, int) bool
	getDrawableCulling                             func(uintptr, int) int
	getDrawableDynamicFlagIsVisible                func(uintptr, int) bool
	getDrawableDynamicFlagVertexPositionsDidChange func(uintptr, int) bool
	getTextureFileName                             func(uintptr, int) string
	update                                         func(uintptr)
	getPixelsPerUnit                               func(uintptr) float32
	getCanvasWidth                                 func(uintptr) float32
	getCanvasHeight                                func(uintptr) float32
	getCanvasWidthPixel                            func(uintptr) float32
	getCanvasHeightPixel                           func(uintptr) float32
	getParameterCount                              func(uintptr) int
	getParameterId                                 func(uintptr, int) string
	addParameterValue                              func(uintptr, int, float32)
}

func NewGame() *Game {
	return &Game{
		textureMap:   make(map[int]*ebiten.Image),
		parameterMap: make(map[string]int),
		surface:      ebiten.NewImage(1, 1),
		maskBuffer:   ebiten.NewImage(1, 1),
		frameBuffer:  ebiten.NewImage(1, 1),
		scaleX:       1,
		scaleY:       1,
		width:        DEFAULT_WIDTH,
		height:       DEFAULT_HEIGHT,
	}
}

func (g *Game) Update() error {
	g.update(g.cubism)
	// マウスに目線を追従させる
	// TODO ウィンドウではなく描画対象のSurfaceを基準にするべきでは？
	if ebiten.IsFocused() {
		x, y := ebiten.CursorPosition()
		if x >= 0 && x < g.width && y >= 0 && y < g.height {
			// xとyを-1~1の範囲に変換する。なお、Y座標は反転させる
			convertedX := float32(x)*2/float32(g.width) - 1
			convertedY := (float32(y)*2/float32(g.height) - 1) * -1
			g.addParameterValue(g.cubism, g.parameterMap["ParamAngleX"], convertedX*30.0)
			g.addParameterValue(g.cubism, g.parameterMap["ParamAngleY"], convertedY*30.0)
			g.addParameterValue(g.cubism, g.parameterMap["ParamAngleZ"], convertedX*convertedY*-30.0)
			g.addParameterValue(g.cubism, g.parameterMap["ParamBodyAngleX"], convertedX*10.0)
			g.addParameterValue(g.cubism, g.parameterMap["ParamEyeBallX"], convertedX)
			g.addParameterValue(g.cubism, g.parameterMap["ParamEyeBallY"], convertedY)
		}
	}

	// projectionPtr := g.update(g.cubism)
	for i, _ := range g.drawables {
		vertexCount := g.getDrawableVertexCount(g.cubism, i)
		g.drawables[i].vertexPositions = unsafe.Slice((*Vertex2)(unsafe.Pointer(g.getDrawableVertexPositions(g.cubism, i))), vertexCount)
		g.drawables[i].vertexUVs = unsafe.Slice((*Vertex2)(unsafe.Pointer(g.getDrawableVertexUVs(g.cubism, i))), vertexCount)
		indexCount := g.getIndexCount(g.cubism, i)
		g.drawables[i].vertexIndices = unsafe.Slice((*uint16)(unsafe.Pointer(g.getDrawableVertexIndices(g.cubism, i))), indexCount)
		g.drawables[i].id = g.getDrawableId(g.cubism, i)
		g.drawables[i].opacity = g.getDrawableOpacity(g.cubism, i)
		g.drawables[i].blendMode = g.getDrawableBlendMode(g.cubism, i)
		maskCount := g.getMaskCount(g.cubism, i)
		g.drawables[i].masks = unsafe.Slice((*int32)(unsafe.Pointer(g.getMasks(g.cubism, i))), maskCount)
		g.drawables[i].isInvertedMask = g.getDrawableIsInvertedMask(g.cubism, i)
		g.drawables[i].culling = g.getDrawableCulling(g.cubism, i)
		g.drawables[i].isVisible = g.getDrawableDynamicFlagIsVisible(g.cubism, i)
		g.drawables[i].vertexPositionsDidChange = g.getDrawableDynamicFlagVertexPositionsDidChange(g.cubism, i)
		g.drawables[i].vertices = make([]ebiten.Vertex, 0)
		for j, pos := range g.drawables[i].vertexPositions {
			// posは-1~1の範囲になっているので、画面サイズに合わせて変換する
			x := (pos.X + 1) * float32(g.surface.Bounds().Dx()) / 2
			y := (pos.Y + 1) * float32(g.surface.Bounds().Dy()) / 2
			// 画像の左上を原点とするため、y座標を反転させる
			y = float32(g.surface.Bounds().Dy()) - y
			// UV座標は0~1の範囲になっているので、画像サイズに合わせて変換する
			uvX := float32(g.drawables[i].vertexUVs[j].X) * float32(g.textureMap[g.drawables[i].textureIndex].Bounds().Dx())
			uvY := float32(g.drawables[i].vertexUVs[j].Y) * float32(g.textureMap[g.drawables[i].textureIndex].Bounds().Dy())
			// 画像の左上を原点とするため、y座標を反転させる
			uvY = float32(g.textureMap[g.drawables[i].textureIndex].Bounds().Dy()) - uvY
			g.drawables[i].vertices = append(g.drawables[i].vertices, ebiten.Vertex{
				DstX:   x,
				DstY:   y,
				SrcX:   uvX,
				SrcY:   uvY,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			})
		}
	}
	rawRenderOrders := unsafe.Slice((*int32)(unsafe.Pointer(g.getDrawableRenderOrders(g.cubism))), len(g.drawables))
	g.renderOrders = make([]int, len(g.drawables))
	for i, order := range rawRenderOrders {
		g.renderOrders[order] = i
	}
	if g.getCanvasWidth(g.cubism) > 1.0 && g.width < g.height {
		g.scaleX = 1.0
		g.scaleY = float64(g.width) / float64(g.height)
	} else {
		// projection.Scale(static_cast<float>(height) / static_cast<float>(width), 1.0f);
		g.scaleX = float64(g.height) / float64(g.width)
		g.scaleY = 1.0
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.White)
	g.surface.Fill(color.RGBA{0, 255, 0, 255})
	for _, order := range g.renderOrders {
		d := g.drawables[order]
		if !d.isVisible || d.opacity == 0 {
			continue
		}
		if len(d.masks) > 0 {
			g.frameBuffer.Fill(color.RGBA{0, 0, 0, 0})
			g.maskBuffer.Fill(color.RGBA{0, 0, 0, 0})
			for _, maskIndex := range d.masks {
				mask := g.drawables[maskIndex]
				maskOptions := &colorm.DrawTrianglesOptions{}
				maskColorM := colorm.ColorM{}
				maskColorM.Scale(0, 0, 0, 1)
				maskOptions.AntiAlias = true
				colorm.DrawTriangles(g.maskBuffer, mask.vertices, mask.vertexIndices, g.textureMap[mask.textureIndex], maskColorM, maskOptions)
			}
			g.frameBuffer.DrawTriangles(d.vertices, d.vertexIndices, g.textureMap[d.textureIndex], &ebiten.DrawTrianglesOptions{})
			options := &ebiten.DrawRectShaderOptions{}
			options.Images[0] = g.maskBuffer
			options.Images[1] = g.frameBuffer
			g.surface.DrawRectShader(g.frameBuffer.Bounds().Dx(), g.frameBuffer.Bounds().Dy(), g.shader, options)
		} else {
			options := &colorm.DrawTrianglesOptions{}
			colorM := colorm.ColorM{}
			colorM.Scale(1, 1, 1, float64(d.opacity))
			options.AntiAlias = true
			colorm.DrawTriangles(g.surface, d.vertices, d.vertexIndices, g.textureMap[d.textureIndex], colorM, options)
		}
	}
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(g.scaleX, g.scaleY)
	// windowに合わせてスケールを調整する
	options.GeoM.Scale(float64(g.width)/float64(g.surface.Bounds().Dx()), float64(g.height)/float64(g.surface.Bounds().Dy()))
	// ニアレストだとジャギる気がするのでリニアにしている
	options.Filter = ebiten.FilterLinear
	screen.DrawImage(g.surface, options)
	ebitenutil.DebugPrint(screen, fmt.Sprintf("TPS: %0.2f\nFPS: %0.2f", ebiten.ActualTPS(), ebiten.ActualFPS()))
}

func (g *Game) Layout(w, h int) (int, int) {
	if g.width != w || g.height != h {
		g.width = w
		g.height = h
	}
	// return g.surface.Bounds().Dx(), g.surface.Bounds().Dy()
	return w, h
}

func main() {
	lib, err := purego.Dlopen("libcubism.dylib", purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err != nil {
		log.Fatal(err)
	}

	g := NewGame()

	purego.RegisterLibFunc(&g.newCubism, lib, "new_cubism")
	purego.RegisterLibFunc(&g.loadModel, lib, "load_model")
	purego.RegisterLibFunc(&g.getDrawableCount, lib, "get_drawable_count")
	purego.RegisterLibFunc(&g.getTextureIndex, lib, "get_texture_index")
	purego.RegisterLibFunc(&g.getDrawableVertexCount, lib, "get_drawable_vertex_count")
	purego.RegisterLibFunc(&g.getDrawableVertexPositions, lib, "get_drawable_vertex_positions")
	purego.RegisterLibFunc(&g.getDrawableVertexUVs, lib, "get_drawable_vertex_uvs")
	purego.RegisterLibFunc(&g.getIndexCount, lib, "get_index_count")
	purego.RegisterLibFunc(&g.getDrawableVertexIndices, lib, "get_drawable_vertex_indices")
	purego.RegisterLibFunc(&g.getDrawableId, lib, "get_drawable_id")
	purego.RegisterLibFunc(&g.getDrawableOpacity, lib, "get_drawable_opacity")
	purego.RegisterLibFunc(&g.getDrawableRenderOrders, lib, "get_drawable_render_orders")
	purego.RegisterLibFunc(&g.getDrawableBlendMode, lib, "get_drawable_blend_mode")
	purego.RegisterLibFunc(&g.getMaskCount, lib, "get_mask_count")
	purego.RegisterLibFunc(&g.getMasks, lib, "get_masks")
	purego.RegisterLibFunc(&g.getDrawableIsInvertedMask, lib, "get_drawable_is_inverted_mask")
	purego.RegisterLibFunc(&g.getDrawableCulling, lib, "get_drawable_culling")
	purego.RegisterLibFunc(&g.getDrawableDynamicFlagIsVisible, lib, "get_drawable_dynamic_flag_is_visible")
	purego.RegisterLibFunc(&g.getDrawableDynamicFlagVertexPositionsDidChange, lib, "get_drawable_dynamic_flag_vertex_positions_did_change")
	purego.RegisterLibFunc(&g.getTextureFileName, lib, "get_texture_file_name")
	purego.RegisterLibFunc(&g.update, lib, "update")
	purego.RegisterLibFunc(&g.getPixelsPerUnit, lib, "get_pixels_per_unit")
	purego.RegisterLibFunc(&g.getCanvasWidth, lib, "get_canvas_width")
	purego.RegisterLibFunc(&g.getCanvasHeight, lib, "get_canvas_height")
	purego.RegisterLibFunc(&g.getCanvasWidthPixel, lib, "get_canvas_width_pixel")
	purego.RegisterLibFunc(&g.getCanvasHeightPixel, lib, "get_canvas_height_pixel")
	purego.RegisterLibFunc(&g.getParameterCount, lib, "get_parameter_count")
	purego.RegisterLibFunc(&g.getParameterId, lib, "get_parameter_id")
	purego.RegisterLibFunc(&g.addParameterValue, lib, "add_parameter_value")

	g.shader, err = ebiten.NewShader(shaderSrc)
	if err != nil {
		log.Fatal(err)
	}
	g.cubism = g.newCubism(ebiten.TPS())
	path, _ := filepath.Abs(filepath.Join("Resources/", NAME, "/"))
	g.loadModel(g.cubism, NAME, path+"/")
	g.surface = ebiten.NewImage(int(g.getCanvasWidthPixel(g.cubism)), int(g.getCanvasHeightPixel(g.cubism)))
	g.maskBuffer = ebiten.NewImage(int(g.getCanvasWidthPixel(g.cubism)), int(g.getCanvasHeightPixel(g.cubism)))
	g.frameBuffer = ebiten.NewImage(int(g.getCanvasWidthPixel(g.cubism)), int(g.getCanvasHeightPixel(g.cubism)))

	drawableCount := g.getDrawableCount(g.cubism)
	g.drawables = make([]Drawable, drawableCount)
	for i := 0; i < drawableCount; i++ {
		g.drawables[i].textureIndex = g.getTextureIndex(g.cubism, i)
		if _, ok := g.textureMap[g.drawables[i].textureIndex]; !ok {
			img, _, err := ebitenutil.NewImageFromFile(g.getTextureFileName(g.cubism, g.drawables[i].textureIndex))
			if err != nil {
				log.Fatal(err)
			}
			g.textureMap[g.drawables[i].textureIndex] = img
		}
		vertexCount := g.getDrawableVertexCount(g.cubism, i)
		g.drawables[i].vertexPositions = unsafe.Slice((*Vertex2)(unsafe.Pointer(g.getDrawableVertexPositions(g.cubism, i))), vertexCount)
		g.drawables[i].vertexUVs = unsafe.Slice((*Vertex2)(unsafe.Pointer(g.getDrawableVertexUVs(g.cubism, i))), vertexCount)
		indexCount := g.getIndexCount(g.cubism, i)
		g.drawables[i].vertexIndices = unsafe.Slice((*uint16)(unsafe.Pointer(g.getDrawableVertexIndices(g.cubism, i))), indexCount)
		g.drawables[i].id = g.getDrawableId(g.cubism, i)
		g.drawables[i].opacity = g.getDrawableOpacity(g.cubism, i)
		g.drawables[i].blendMode = g.getDrawableBlendMode(g.cubism, i)
		maskCount := g.getMaskCount(g.cubism, i)
		g.drawables[i].masks = unsafe.Slice((*int32)(unsafe.Pointer(g.getMasks(g.cubism, i))), maskCount)
		g.drawables[i].isInvertedMask = g.getDrawableIsInvertedMask(g.cubism, i)
		g.drawables[i].culling = g.getDrawableCulling(g.cubism, i)
		g.drawables[i].isVisible = g.getDrawableDynamicFlagIsVisible(g.cubism, i)
		g.drawables[i].vertices = make([]ebiten.Vertex, 0)
		for j, pos := range g.drawables[i].vertexPositions {
			// posは-1~1の範囲になっているので、画面サイズに合わせて変換する
			x := (pos.X + 1) * float32(g.surface.Bounds().Dx()) / 2
			y := (pos.Y + 1) * float32(g.surface.Bounds().Dy()) / 2
			// 画像の左上を原点とするため、y座標を反転させる
			y = float32(g.surface.Bounds().Dy()) - y
			// UV座標は0~1の範囲になっているので、画像サイズに合わせて変換する
			uvX := float32(g.drawables[i].vertexUVs[j].X) * float32(g.textureMap[g.drawables[i].textureIndex].Bounds().Dx())
			uvY := float32(g.drawables[i].vertexUVs[j].Y) * float32(g.textureMap[g.drawables[i].textureIndex].Bounds().Dy())
			// 画像の左上を原点とするため、y座標を反転させる
			uvY = float32(g.textureMap[g.drawables[i].textureIndex].Bounds().Dy()) - uvY
			g.drawables[i].vertices = append(g.drawables[i].vertices, ebiten.Vertex{
				DstX:   x,
				DstY:   y,
				SrcX:   uvX,
				SrcY:   uvY,
				ColorR: 1,
				ColorG: 1,
				ColorB: 1,
				ColorA: 1,
			})
		}
	}
	rawRenderOrders := unsafe.Slice((*int32)(unsafe.Pointer(g.getDrawableRenderOrders(g.cubism))), drawableCount)
	g.renderOrders = make([]int, drawableCount)
	for i, order := range rawRenderOrders {
		g.renderOrders[order] = i
	}
	parameterCount := g.getParameterCount(g.cubism)
	for i := 0; i < parameterCount; i++ {
		g.parameterMap[g.getParameterId(g.cubism, i)] = i
	}

	ebiten.SetWindowTitle("Ebitengine Live2D Demo")
	ebiten.SetWindowSize(DEFAULT_WIDTH, DEFAULT_HEIGHT)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
