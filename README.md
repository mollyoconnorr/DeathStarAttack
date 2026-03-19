# Death Star Attack (Go + Pixel)

A fast-paced 2D arcade-style shooter built in **Go** using the **Pixel** library for graphics and **Beep** for sound. Take control of a rebel pilot to destroy the Death Star while dodging enemy fire and asteroids.

---

## Game Features

* **Player-controlled ship** with LEFT/RIGHT movement and SPACE to shoot lasers.
* **Enemies** move across the screen and fire bullets at varying difficulty.
* **Asteroids** spawn randomly and add challenge to the battlefield.
* **Three escalating levels** of difficulty.
* **Starfield background** with scrolling perspective effect.
* **Opening Star Wars crawl** for cinematic effect.
* **Multiple background music tracks** for immersion.
* Lives, score, and level tracking displayed in the window title.

---

## Requirements

* **Go 1.20+** installed
  (Download: [https://golang.org/dl/](https://golang.org/dl/))
* **Pixel library** installed (`github.com/faiface/pixel`)
* **Beep library** for audio (`github.com/faiface/beep`, `mp3`)
* **Basic font** support (`golang.org/x/image/font/basicfont`)

These can be installed using Go Modules (see Installation below).

---

## Installation

1. **Clone the repository**

```bash
git clone https://github.com/mollyoconnorr/DeathStarAttack.git
cd DeathStarAttack
```

2. **Install dependencies via Go Modules**

```bash
go mod tidy
```

3. **Run the game**

```bash
go run main.go
```

> Make sure your terminal is in the same directory as `main.go` and the `music` folder containing mp3 files.

---

## How to Play

1. The game starts with a **Star Wars-style opening crawl**. Press **ENTER** to begin.
2. Use **LEFT** and **RIGHT** arrows to move your ship.
3. Press **SPACE** to fire lasers.
4. Avoid enemy bullets and asteroids.
5. Destroy the enemy ship by hitting it enough times to complete the level.
6. Survive **3 lives**. If lives reach 0, the game ends.
7. Complete all 3 levels to destroy the Death Star.

---

## Math & Mechanics Behind the Game

* **Starfield movement:**
  Stars move downward with a small velocity to simulate motion through space.

  ```go
  stars[i].Y -= 1 // slower for depth, faster for closer stars
  ```
* **Enemy movement:**
  Enemy ship moves horizontally and reverses direction at window edges:

  ```go
  enemy.X += enemyDir * enemySpeed
  if enemy.X-enemySize/2 <= 0 || enemy.X+enemySize/2 >= 800 {
      enemyDir *= -1
  }
  ```
* **Laser and bullet collisions:**
  Checks if positions overlap within rectangular bounding boxes.
* **Opening crawl perspective:**
  Scale calculated from height for perspective effect:

  ```go
  scale := 2.0 - (crawlY / 600.0)
  ```
* **Level scaling:**
  Enemy speed, firing rate, and asteroid spawn probability increase with levels.

---

## Cool Features

* Star Wars-style **opening crawl** with scaling perspective.
* **Dynamic background stars** for depth.
* **Multiple levels of difficulty**, with escalating enemy fire patterns.
* **Randomly spawning asteroids** to make gameplay unpredictable.
* **Background music** changes per level.

---

## Reference

* Music control functions (`playMusic` and `stopMusic`) were implemented with help from **ChatGPT**.
* All music was downloaded from https://archive.org/ 

---

## Customization

* Add or remove **asteroids** or tweak **enemy behavior** by editing:

  * `enemySpeed`, `enemyShotInterval`, `bulletsPerShot`, `asteroidSpawnRate`
* Adjust **number of stars** in the background by modifying the `stars` slice.
* Replace music by updating the **music folder** and file names in `playMusic`.
* Modify **player speed** or **laser speed** in variables:

  * `playerSpeed` and `lasers[i].pos.Y += 8`

---
