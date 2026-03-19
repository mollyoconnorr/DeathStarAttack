package main

/*
===============================================
           Death Star Attack Game
===============================================

Author: [Your Name]
Date:   [Today's Date]
Language: Go (Golang)
Libraries: Pixel (pixelgl, imdraw, text), Beep (audio), basicfont, colornames

Description:
-------------
"Death Star Attack" is a 2D space shooter game inspired by Star Wars.
The player controls a rebel starship attempting to destroy the Death Star
while avoiding enemy fire and falling asteroids.

Gameplay Features:
------------------
- Star Wars-style opening crawl with animated perspective text.
- Multiple levels (3), each increasing in difficulty:
    - Faster enemies
    - More enemy bullets per shot
    - More frequent asteroids
- Player ship can move left/right and shoot lasers.
- Collision detection between player, enemy, and asteroids.
- Lives and score tracking.
- Victory and Game Over screens.
- Background music and sound effects for immersion.

Controls:
---------
- LEFT / RIGHT Arrow Keys: Move player ship horizontally
- SPACE: Fire laser
- ENTER: Start game or advance past screens

Technical Details:
------------------
- The game window is 800x600 pixels.
- Uses Pixel library for rendering shapes, text, and graphics.
- Uses Beep library for background music and looping.
- Stars and text animation simulate a parallax effect.
- Enemy shooting angles use simple linear math to spread bullets.
- All game objects (player, lasers, enemy shots, asteroids) are represented
  as structs with position and movement properties.

Mathematical/Technical Notes:
-----------------------------
- Enemy bullet spread: When multiple bullets are fired, each bullet's X-velocity
  is calculated as:
      angle := float64(i-(bulletsPerShot-1)/2) * 2
  This spreads bullets symmetrically around the center.
- Starfield animation: Star Y-positions decrease over time for downward motion,
  looping back to the top when reaching Y < 0.
- Perspective effect for opening crawl: Scale factor decreases as crawlY
  increases, simulating depth:
      scale := 2.0 - (crawlY / 600.0)
  Minimum scale is clamped to 0.3 to avoid inversion.
- Collision detection: Uses axis-aligned bounding box (AABB) checks for
  rectangles (player, enemies, asteroids, lasers).

Usage:
------
Run the program using:
    go run main.go
Make sure music files are in a "music/" folder:
    - starwars.mp3
    - the-falcon.mp3
    - march-resistence.mp3
    - imperial-march.mp3
    - across-the-stars.mp3
    - cantina-band.mp3

This program demonstrates:
- Basic 2D game loop structure in Go
- Keyboard input handling
- Sprite and text rendering with Pixel
- Simple physics and collision detection
- Background music with Beep

===============================================
*/
import (
	"image/color"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/pixel"
	"github.com/faiface/pixel/imdraw"
	"github.com/faiface/pixel/pixelgl"
	"github.com/faiface/pixel/text"
	"golang.org/x/image/colornames"
	"golang.org/x/image/font/basicfont"
)

// Vec2 is an alias for pixel.Vec, which represents a 2D vector (X, Y).
// We use this for positions, velocities, and movements throughout the game.
type Vec2 = pixel.Vec

// --- Game objects ---

// Laser represents a single player-fired laser.
// 'pos' is the current position of the laser in 2D space.
type Laser struct{ pos Vec2 }

// EnemyShot represents a single shot fired by the enemy.
// 'pos' is the current position, 'vel' is the velocity vector (X, Y) that
// determines how the shot moves each frame.
type EnemyShot struct{ pos, vel Vec2 }

// Asteroid represents a falling obstacle in the game.
// 'pos' is the current position, 'size' determines the width/height (square),
// and 'speed' determines how fast the asteroid moves down each frame.
type Asteroid struct {
	pos   Vec2
	size  float64
	speed float64
}

// --- Music control ---

// currentCtrl allows us to pause/resume/stop the music.
// currentStreamer is the audio stream that we decode and play.
var (
	currentCtrl     *beep.Ctrl
	currentStreamer beep.StreamSeekCloser
)

// playMusic loads and plays a music file. If 'loop' is true, it loops indefinitely.
func playMusic(filename string, loop bool) {
	// Pause any currently playing music
	if currentCtrl != nil {
		speaker.Lock() // Lock speaker to avoid audio glitches
		currentCtrl.Paused = true
		speaker.Unlock()
	}
	if currentStreamer != nil {
		currentStreamer.Close() // Close previous audio stream to free resources
	}

	// Open the MP3 file
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	// Decode MP3 into a Streamer and get the audio format
	streamer, format, err := mp3.Decode(f)
	if err != nil {
		panic(err)
	}

	currentStreamer = streamer

	// Initialize speaker if not already initialized
	// SampleRate.N(time.Second/10) sets the buffer size to 1/10 second
	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	// If looping is requested, wrap the streamer in a loop
	var toPlay beep.Streamer = streamer
	if loop {
		// -1 indicates infinite looping
		toPlay = beep.Loop(-1, streamer)
	}

	// Wrap the streamer in a Ctrl struct so we can pause or stop it later
	ctrl := &beep.Ctrl{Streamer: toPlay, Paused: false}
	currentCtrl = ctrl

	// Play the music asynchronously
	speaker.Play(ctrl)
}

// stopMusic immediately stops the current music and releases resources
func stopMusic() {
	if currentCtrl != nil {
		speaker.Lock() // Lock the speaker before pausing
		currentCtrl.Paused = true
		speaker.Unlock()
	}
	if currentStreamer != nil {
		currentStreamer.Close() // Close the streamer to free memory
		currentStreamer = nil
	}
}

// --- Main game loop ---
func run() {
	// Start opening crawl music
	go playMusic("music/starwars.mp3", true)

	// --- Window setup ---
	cfg := pixelgl.WindowConfig{
		Title:  "Death Star Attack",
		Bounds: pixel.R(0, 0, 800, 600),
		VSync:  true,
	}
	win, _ := pixelgl.NewWindow(cfg)

	rand.Seed(time.Now().UnixNano()) // random seed

	// --- Game variables ---
	crawlY := -200.0
	crawlSpeed := 25.0
	player := Vec2{X: 400, Y: 50}
	playerSize := 20.0
	playerSpeed := 5.0
	lives := 3
	level := 1
	score := 0
	targetScore := 15

	showInstructions := true
	gameOver := false
	waitingForNextLevel := false
	wonGame := false

	var lasers []Laser
	var enemyShots []EnemyShot
	var asteroids []Asteroid

	enemy := Vec2{X: 400, Y: 550}
	enemySize := 80.0
	enemyDir := 1.0
	lastEnemyShot := time.Now()

	// --- Stars for background ---
	stars := []Vec2{}
	for i := 0; i < 100; i++ {
		// Initialize 100 stars at random positions:
		// X coordinate: random float between 0 and 800 (screen width)
		// Y coordinate: random float between 0 and 600 (screen height)
		// rand.Float64() generates a float in [0.0, 1.0), multiplied by screen size to scale it
		stars = append(stars, Vec2{X: rand.Float64() * 800, Y: rand.Float64() * 600})
	}

	lastShotTime := time.Now()
	// Create a text atlas using a basic fixed-width font (7x13 pixels per character)
	// ASCII characters only, which allows us to efficiently render text on screen
	atlas := text.NewAtlas(basicfont.Face7x13, text.ASCII)
	lastTime := time.Now()

	// --- Main loop ---
	for !win.Closed() {
		win.Clear(colornames.Black)

		// --- Instruction screen ---
		if showInstructions {
			now := time.Now()
			dt := now.Sub(lastTime).Seconds()
			lastTime = now
			win.Clear(color.Black)

			// Draw background stars (slower for instruction screen)
			imdBg := imdraw.New(nil)
			imdBg.Color = color.RGBA{R: 100, G: 100, B: 255, A: 80} // faint blue stars
			for i := range stars {
				stars[i].Y -= 0.3 // slow movement
				// If the star moves off the bottom of the screen, wrap it to the top
				// Reset Y to 600 (screen height) and assign a new random X (0 to 800)
				// This creates an endless looping starfield
				if stars[i].Y < 0 {
					stars[i].Y = 600
					stars[i].X = rand.Float64() * 800
				}
				imdBg.Push(stars[i])
				imdBg.Push(stars[i].Add(pixel.V(2, 2)))
				imdBg.Rectangle(0)
			}
			imdBg.Draw(win)

			// --- Crawl text ---
			crawlText := `A long time ago in a galaxy far, far away....

			DEATH STAR ATTACK
			
			The Empire has completed the
			ultimate weapon: the Death Star.
			
			A lone rebel pilot stands as
			the last hope against total destruction.
			
			You must survive relentless enemy fire,
			dodge deadly asteroids, and strike back.
			
			It will take 15 successful hits
			to destroy the Death Star.
			
			There are 3 escalating levels
			of increasing difficulty.
			
			You have only 3 lives.
			
			Use LEFT and RIGHT to navigate.
			Press SPACE to fire your lasers.
			
			Restore hope to the galaxy...
			
			Press ENTER to begin...`

			txt := text.New(pixel.V(-200, 0), atlas)
			txt.Color = color.RGBA{255, 220, 0, 255} // Star Wars yellow
			txt.WriteString(crawlText)

			// --- Animate upward ---
			// Move the crawl text up each frame based on crawlSpeed and delta time (dt)
			// dt is the time difference since last frame, ensuring smooth animation regardless of frame rate
			crawlY += crawlSpeed * dt
			if crawlY > 600 {
				crawlY = -400
			}

			// --- Perspective effect ---
			// Scale factor decreases as crawlY increases to simulate 3D perspective (further away = smaller)
			// scale = 2.0 - (crawlY / 600.0)
			//   When crawlY = 0 -> scale = 2.0 (large, near bottom)
			//   When crawlY = 600 -> scale = 1.0 (smaller, moving upward)
			// Clamp scale so it never becomes too small
			center := pixel.V(450, 300)
			scale := 2.0 - (crawlY / 600.0) // higher = smaller
			if scale < 0.3 {
				scale = 0.3
			}
			mat := pixel.IM.Moved(center).Scaled(center, scale).Moved(pixel.V(0, crawlY))
			txt.Draw(win, mat)

			win.Update()

			// Start game on Enter
			if win.JustPressed(pixelgl.KeyEnter) {
				showInstructions = false
				// start first level song
				stopMusic()
				go playMusic("music/the-falcon.mp3", true)
				win.SetTitle("Death Star Attack - Level 1")
			}

			continue
		}

		// --- Game over screen ---
		if gameOver {
			win.Clear(colornames.Black)
			center := win.Bounds().Center()

			// Draw "GAME OVER" title
			title := text.New(center, atlas)
			title.Color = colornames.Red
			title.WriteString("GAME OVER")
			tb := title.Bounds()
			title.Draw(win, pixel.IM.Moved(center.Sub(tb.Center())).Scaled(center, 2))

			// Draw subtext
			sub := text.New(center, atlas)
			sub.Color = colornames.White
			sub.WriteString("Press ENTER to restart")
			subB := sub.Bounds()
			sub.Draw(win, pixel.IM.Moved(center.Sub(subB.Center()).Add(pixel.V(0, -60))))

			win.Update()

			if win.JustPressed(pixelgl.KeyEnter) {
				// Reset game variables
				lives = 3
				level = 1
				score = 0
				lasers = nil
				enemyShots = nil
				asteroids = nil
				player.X = 400
				gameOver = false
				// start new game music
				stopMusic()
				go playMusic("music/starwars.mp3", true)
				crawlY = -300.0
				showInstructions = true
			}

			continue
		}

		// --- Level complete screen ---
		if waitingForNextLevel {
			win.Clear(colornames.Black)
			center := win.Bounds().Center()

			// Title
			title := text.New(center, atlas)
			title.Color = colornames.Yellow
			title.WriteString("LEVEL " + strconv.Itoa(level) + " COMPLETE")
			tb := title.Bounds()
			title.Draw(win, pixel.IM.Moved(center.Sub(tb.Center())).Scaled(center, 2))

			// Subtext
			sub := text.New(center, atlas)
			sub.Color = colornames.White
			sub.WriteString("Press ENTER to continue")
			sb := sub.Bounds()
			sub.Draw(win, pixel.IM.Moved(center.Sub(sb.Center()).Add(pixel.V(0, -40))))

			win.Update()

			if win.JustPressed(pixelgl.KeyEnter) {
				// Prepare next level
				level++
				score = 0
				lasers = nil
				enemyShots = nil
				asteroids = nil
				enemy.X = 400
				enemyDir = 1.0
				lastEnemyShot = time.Now()
				waitingForNextLevel = false
				// match song with corresponding level
				stopMusic()
				switch level {
				case 1:
					go playMusic("music/the-falcon.mp3", true)
				case 2:
					go playMusic("music/march-resistence.mp3", true)
				case 3:
					go playMusic("music/imperial-march.mp3", true)
				default:
					go playMusic("music/starwars.mp3", true)
				}
			}

			continue
		}

		// --- Victory screen ---
		if wonGame {
			win.Clear(colornames.Black)
			center := win.Bounds().Center()

			// Title
			title := text.New(center, atlas)
			title.Color = colornames.Green
			title.WriteString("THE DEATH STAR HAS BEEN DESTROYED")
			tb := title.Bounds()
			title.Draw(win, pixel.IM.Moved(center.Sub(tb.Center())).Scaled(center, 2))

			// Score text
			scoreTxt := text.New(pixel.ZV, atlas)
			scoreTxt.Color = colornames.White
			scoreTxt.WriteString("THE GALAXY THANKS YOU")
			sb := scoreTxt.Bounds()
			scoreTxt.Draw(win, pixel.IM.Moved(center.Sub(sb.Center()).Add(pixel.V(0, -30))))

			// Subtext
			sub := text.New(pixel.ZV, atlas)
			sub.Color = colornames.White
			sub.WriteString("Press ENTER to restart")
			subB := sub.Bounds()
			sub.Draw(win, pixel.IM.Moved(center.Sub(subB.Center()).Add(pixel.V(0, -60))))

			win.Update()

			if win.JustPressed(pixelgl.KeyEnter) {
				// Reset game
				lives = 3
				level = 1
				score = 0
				lasers = nil
				enemyShots = nil
				asteroids = nil
				player.X = 400
				wonGame = false
				showInstructions = true
				crawlY = -300.0
			}

			continue
		}

		// --- Difficulty settings per level ---
		var enemySpeed float64
		var enemyShotInterval time.Duration
		var bulletsPerShot int
		var asteroidSpawnRate float64
		switch level {
		case 1:
			enemySpeed = 2.0
			enemyShotInterval = 600 * time.Millisecond
			bulletsPerShot = 1
			asteroidSpawnRate = 0.01
		case 2:
			enemySpeed = 3.0
			enemyShotInterval = 500 * time.Millisecond
			bulletsPerShot = 2
			asteroidSpawnRate = 0.012
		case 3:
			enemySpeed = 3.0
			enemyShotInterval = 500 * time.Millisecond
			bulletsPerShot = 3
			asteroidSpawnRate = 0.014
		}

		// --- Player input ---
		if win.Pressed(pixelgl.KeyLeft) && player.X-playerSize/2 > 0 {
			player.X -= playerSpeed
		}
		if win.Pressed(pixelgl.KeyRight) && player.X+playerSize/2 < 800 {
			player.X += playerSpeed
		}
		if win.Pressed(pixelgl.KeySpace) && time.Since(lastShotTime) > 200*time.Millisecond {
			lasers = append(lasers, Laser{pos: Vec2{X: player.X, Y: player.Y + playerSize}})
			lastShotTime = time.Now()
		}

		// --- Move lasers ---
		for i := range lasers {
			lasers[i].pos.Y += 8
		}

		// --- Move enemy shots ---
		for i := range enemyShots {
			enemyShots[i].pos = enemyShots[i].pos.Add(enemyShots[i].vel)
		}

		// --- Spawn enemy shots ---
		// Spawn enemy shots at regular intervals
		if time.Since(lastEnemyShot) > enemyShotInterval {

			for i := 0; i < bulletsPerShot; i++ {
				var vel Vec2

				if bulletsPerShot == 1 {
					// Single bullet goes straight down
					vel = Vec2{X: 0, Y: -5}
				} else {
					// Spread multiple bullets horizontally
					// i ranges from 0 to bulletsPerShot-1
					// Center the spread around 0 using: i - (bulletsPerShot-1)/2
					// Multiply by 2 to adjust horizontal spacing
					angle := float64(i-(bulletsPerShot-1)/2) * 2
					vel = Vec2{X: angle, Y: -5} // downward velocity with slight horizontal offset
				}

				// Append new enemy shot at enemy's current position, moving with calculated velocity
				enemyShots = append(enemyShots, EnemyShot{
					pos: Vec2{X: enemy.X, Y: enemy.Y - enemySize/2},
					vel: vel,
				})
			}

			// Reset the timer for the next shot
			lastEnemyShot = time.Now()
		}

		// --- Move enemy ---
		// Update enemy horizontal position: position += direction * speed
		// Reverse direction if hitting the left (0) or right (800) edges
		enemy.X += enemyDir * enemySpeed
		if enemy.X-enemySize/2 <= 0 || enemy.X+enemySize/2 >= 800 {
			enemyDir *= -1 // flip direction (simple reflection)
		}

		// --- Spawn asteroids ---
		// Randomly spawn an asteroid based on asteroidSpawnRate probability each frame
		if rand.Float64() < asteroidSpawnRate {
			asteroids = append(asteroids, Asteroid{
				// X: random horizontal position between 0 and 800
				// Y: start at top of screen (600)
				pos: Vec2{X: rand.Float64() * 800, Y: 600},
				// Size: random between 20 and 50 (20 + rand*30)
				size: 20 + rand.Float64()*30,
				// Speed: random downward speed between 2 and 5 (2 + rand*3)
				speed: 2 + rand.Float64()*3,
			})
		}

		// --- Move asteroids ---
		newAsteroids := []Asteroid{}
		for _, a := range asteroids {
			a.pos.Y -= a.speed
			if a.pos.Y+a.size > 0 {
				newAsteroids = append(newAsteroids, a)
			}
		}
		asteroids = newAsteroids

		// --- Draw background stars ---
		imdBg := imdraw.New(nil)
		imdBg.Color = color.RGBA{R: 100, G: 100, B: 255, A: 80}
		for i := range stars {
			stars[i].Y -= 1
			if stars[i].Y < 0 {
				stars[i].Y = 600
				stars[i].X = rand.Float64() * 800
			}
			imdBg.Push(stars[i])
			imdBg.Push(stars[i].Add(pixel.V(2, 2)))
			imdBg.Rectangle(0)
		}
		imdBg.Draw(win)

		// --- Draw player ---
		imdPlayer := imdraw.New(nil)
		imdPlayer.Color = color.RGBA{R: 50, G: 200, B: 255, A: 255}
		imdPlayer.Push(player.Sub(pixel.V(playerSize/2, playerSize/2)))
		imdPlayer.Push(player.Add(pixel.V(playerSize/2, playerSize/2)))
		imdPlayer.Rectangle(0)
		imdPlayer.Draw(win)

		// --- Draw lasers ---
		// Create a new drawing layer for lasers
		imdLaser := imdraw.New(nil)
		imdLaser.Color = color.RGBA{R: 255, G: 50, B: 50, A: 255} // bright red

		for _, l := range lasers {
			// Draw each laser as a small vertical rectangle
			// l.pos is the bottom-left corner; add (4,12) to get top-right
			// This means each laser is 4 pixels wide and 12 pixels tall
			imdLaser.Push(l.pos)
			imdLaser.Push(l.pos.Add(pixel.V(4, 12)))
			imdLaser.Rectangle(0) // draw filled rectangle
		}

		// Render all lasers on the window
		imdLaser.Draw(win)

		// --- Draw enemy ---
		imdEnemy := imdraw.New(nil)
		imdEnemy.Color = color.RGBA{R: 200, G: 200, B: 200, A: 255}
		imdEnemy.Push(enemy.Sub(pixel.V(enemySize/2, enemySize/2)))
		imdEnemy.Push(enemy.Add(pixel.V(enemySize/2, enemySize/2)))
		imdEnemy.Rectangle(0)
		imdEnemy.Draw(win)

		// --- Draw enemy shots ---
		imdEnemyShot := imdraw.New(nil)
		imdEnemyShot.Color = color.RGBA{R: 255, G: 255, B: 50, A: 255}
		for _, es := range enemyShots {
			imdEnemyShot.Push(es.pos)
			imdEnemyShot.Push(es.pos.Add(pixel.V(6, 12)))
			imdEnemyShot.Rectangle(0)
		}
		imdEnemyShot.Draw(win)

		// --- Draw asteroids ---
		imdAst := imdraw.New(nil)
		imdAst.Color = color.RGBA{R: 150, G: 100, B: 50, A: 255}
		for _, a := range asteroids {
			imdAst.Push(a.pos)
			imdAst.Push(a.pos.Add(pixel.V(a.size, a.size)))
			imdAst.Rectangle(0)
		}
		imdAst.Draw(win)

		// --- Collision detection ---
		// Check each laser to see if it hits the enemy
		newLasers := []Laser{}
		for _, l := range lasers {
			// Check if laser is within the enemy's bounding box
			// Enemy box extends from (enemy.X - enemySize/2, enemy.Y - enemySize/2)
			// to (enemy.X + enemySize/2, enemy.Y + enemySize/2)
			if l.pos.X > enemy.X-enemySize/2 && l.pos.X < enemy.X+enemySize/2 &&
				l.pos.Y > enemy.Y-enemySize/2 && l.pos.Y < enemy.Y+enemySize/2 {
				score++  // laser hit the enemy, increment score
				continue // skip adding this laser to new list (laser disappears)
			}
			// Laser did not hit enemy, keep it in the game
			newLasers = append(newLasers, l)
		}
		// Replace the lasers slice with the updated list (removes hit lasers)
		lasers = newLasers

		// --- Player hit detection ---
		// Check if the player collides with any enemy shots or asteroids
		hit := false

		// Check collision with enemy shots
		for _, es := range enemyShots {
			// Enemy shot bounding box check:
			// Player box extends from (player.X - playerSize/2, player.Y - playerSize/2)
			// to (player.X + playerSize/2, player.Y + playerSize/2)
			if es.pos.X > player.X-playerSize/2 && es.pos.X < player.X+playerSize/2 &&
				es.pos.Y > player.Y-playerSize/2 && es.pos.Y < player.Y+playerSize/2 {
				hit = true // Player is hit by a laser
			}
		}

		// Check collision with asteroids
		for _, a := range asteroids {
			// Asteroid box extends from (a.pos.X, a.pos.Y) to (a.pos.X + a.size, a.pos.Y + a.size)
			// Player box check as above
			if player.X+playerSize/2 > a.pos.X && player.X-playerSize/2 < a.pos.X+a.size &&
				player.Y+playerSize/2 > a.pos.Y && player.Y-playerSize/2 < a.pos.Y+a.size {
				hit = true // Player is hit by an asteroid
			}
		}

		if hit {
			lives--
			player.X = 400
			player.Y = 50
			lasers = nil
			enemyShots = nil
			asteroids = nil
			enemy.X = 400
			enemyDir = 1.0
			lastEnemyShot = time.Now()
			if lives <= 0 {
				stopMusic()
				go playMusic("music/across-the-stars.mp3", true)
				gameOver = true
			}
		}

		// --- Level completion check ---
		if score >= targetScore && !waitingForNextLevel {
			if level < 3 {
				stopMusic()
				go playMusic("music/cantina-band.mp3", true)
				waitingForNextLevel = true
			} else {
				stopMusic()
				go playMusic("music/starwars.mp3", true)
				wonGame = true
			}
		}

		// --- Update window title ---
		if !gameOver && !waitingForNextLevel {
			win.SetTitle("Level " + strconv.Itoa(level) + " | Score: " + strconv.Itoa(score) + " | Lives: " + strconv.Itoa(lives))
		}

		win.Update()
		time.Sleep(time.Millisecond * 16) // ~60 FPS
	}
}

func main() {
	pixelgl.Run(run)
}
