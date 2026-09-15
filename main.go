package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ========================================================================
// КОНСТАНТЫ — СОСТОЯНИЯ ИГРЫ
// ========================================================================
const (
	StateStart = iota
	StatePlaying
	StatePaused
	StateGameOver
)

// ========================================================================
// КОНСТАНТЫ — РАЗМЕРЫ ПОЛЯ И СМЕЩЕНИЯ
// ========================================================================
const gameWidth = 40
const gameHeight = 20
const offsetX = 0
const offsetY = 3

// Длительность неуязвимости после удара
const invulnerableDuration = 1500 * time.Millisecond

// Максимальное здоровье игрока
const maxHealth = 3

// ========================================================================
// ТИПЫ МОНЕТ
// ========================================================================
type CoinType int

const (
	CoinNormal CoinType = iota // Обычная: 1 очко
	CoinGold                   // Золотая: 5 очков
	CoinRare                   // Редкая: 10 очков
)

// Points — возвращает количество очков за монету данного типа.
func (ct CoinType) Points() int {
	switch ct {
	case CoinNormal:
		return 1
	case CoinGold:
		return 5
	case CoinRare:
		return 10
	default:
		return 1
	}
}

// randomCoinType — возвращает случайный тип монеты.
// Обычная: 70%, Золотая: 20%, Редкая: 10%
func randomCoinType() CoinType {
	r := rand.Intn(100)
	switch {
	case r < 70:
		return CoinNormal
	case r < 90:
		return CoinGold
	default:
		return CoinRare
	}
}

// ========================================================================
// ТИПЫ БОНУСОВ
// ========================================================================
type PowerUpType int

const (
	PowerUpHeart PowerUpType = iota // +1 жизнь
)

// Char — возвращает символ бонуса.
func (p PowerUpType) Char() rune {
	switch p {
	case PowerUpHeart:
		return '♥'
	default:
		return '?'
	}
}

// Color — возвращает цвет бонуса.
func (p PowerUpType) Color() tcell.Color {
	switch p {
	case PowerUpHeart:
		return tcell.ColorRed
	default:
		return tcell.ColorWhite
	}
}

// ========================================================================
// СТРУКТУРЫ
// ========================================================================

// PowerUp — бонус на игровом поле
type PowerUp struct {
	X, Y int
	Type PowerUpType
}

// Particle — частица (визуальный эффект)
type Particle struct {
	X, Y  int
	Char  rune
	Style tcell.Style
	DirY  int // Направление движения (-1 вверх)
}

// ========================================================================
// ПЕРЕМЕННЫЕ — СИМВОЛЫ ДЛЯ РАМКИ
// ========================================================================
var (
	borderH    = '─'
	borderV    = '│'
	cornerTopL = '┌'
	cornerTopR = '┐'
	cornerBotL = '└'
	cornerBotR = '┘'
)

// ========================================================================
// ПЕРЕМЕННЫЕ — ЦВЕТА
// ========================================================================
var (
	colorPlayer = tcell.ColorGreen
	colorEnemy  = tcell.ColorRed
	colorChaser = tcell.ColorPurple // Цвет умного врага
	colorBorder = tcell.ColorWhite
	colorUI     = tcell.ColorAqua
	colorHealth = tcell.ColorRed
)

// Символы для разных типов монет
var coinTypeChars = map[CoinType]rune{
	CoinNormal: '0',
	CoinGold:   '$',
	CoinRare:   '♦',
}

// Базовые цвета для разных типов монет
var coinTypeBaseColors = map[CoinType]tcell.Color{
	CoinNormal: tcell.ColorYellow,
	CoinGold:   tcell.ColorOrange,
	CoinRare:   tcell.ColorPurple,
}

// ========================================================================
// АНИМАЦИЯ МОНЕТ — уникальная для каждого типа
// ========================================================================

// Обычная монета (жёлтая): 0 → O → o → * → o → O
var coinNormalFrames = []rune{'0', 'O', 'o', '*', 'o', 'O'}
var coinNormalColors = []tcell.Color{
	tcell.ColorYellow,
	tcell.ColorWhite,
	tcell.ColorYellow,
	tcell.ColorOrange,
	tcell.ColorYellow,
	tcell.ColorWhite,
}

// Золотая монета (оранжевая): $ → ¤ → $ → ✦ → $ → ¤
var coinGoldFrames = []rune{'$', '¤', '$', '✦', '$', '¤'}
var coinGoldColors = []tcell.Color{
	tcell.ColorOrange,
	tcell.ColorYellow,
	tcell.ColorOrange,
	tcell.ColorWhite,
	tcell.ColorOrange,
	tcell.ColorYellow,
}

// Редкая монета (пурпурная): ♦ → ◆ → ◇ → ✧ → ◇ → ◆
var coinRareFrames = []rune{'♦', '◆', '◇', '✧', '◇', '◆'}
var coinRareColors = []tcell.Color{
	tcell.ColorPurple,
	tcell.ColorFuchsia,
	tcell.ColorPurple,
	tcell.ColorWhite,
	tcell.ColorPurple,
	tcell.ColorFuchsia,
}

// ========================================================================
// ТИП Coin — МОНЕТА
// ========================================================================
type Coin struct {
	Char  rune
	X, Y  int
	DirX  int
	DirY  int
	Style tcell.Style
	Type  CoinType
}

func NewCoin(char rune, x, y, dirX, dirY int, coinType CoinType) *Coin {
	return &Coin{
		Char:  char,
		X:     x,
		Y:     y,
		DirX:  dirX,
		DirY:  dirY,
		Style: tcell.StyleDefault.Foreground(coinTypeBaseColors[coinType]),
		Type:  coinType,
	}
}

// Update — обновляет позицию монеты за один шаг.
func (c *Coin) Update() {
	nx := c.X + c.DirX
	ny := c.Y + c.DirY
	if nx <= 1 || nx >= gameWidth-2 {
		c.DirX = -c.DirX
	} else {
		c.X = nx
	}
	if ny <= 1 || ny >= gameHeight-2 {
		c.DirY = -c.DirY
	} else {
		c.Y = ny
	}
}

// ========================================================================
// ТИП Enemy — ВРАГ (С ИНТЕЛЛЕКТОМ)
// ========================================================================
type Enemy struct {
	Char  rune
	X, Y  int
	DirX  int
	DirY  int
	Style tcell.Style
	Smart bool // Умный враг преследует игрока
}

func NewEnemy(char rune, x, y, dirX, dirY int, smart bool) *Enemy {
	style := tcell.StyleDefault.Foreground(colorEnemy)
	if smart {
		style = tcell.StyleDefault.Foreground(colorChaser)
	}
	return &Enemy{
		Char:  char,
		X:     x,
		Y:     y,
		DirX:  dirX,
		DirY:  dirY,
		Style: style,
		Smart: smart,
	}
}

// Update — обновляет позицию врага.
func (e *Enemy) Update(playerX, playerY int) {
	if e.Smart {
		e.ChasePlayer(playerX, playerY)
		return
	}

	nx := e.X + e.DirX
	ny := e.Y + e.DirY
	if nx <= 1 || nx >= gameWidth-2 {
		e.DirX = -e.DirX
	} else {
		e.X = nx
	}
	if ny <= 1 || ny >= gameHeight-2 {
		e.DirY = -e.DirY
	} else {
		e.Y = ny
	}
}

// ChasePlayer — преследование игрока с погрешностью 15%.
func (e *Enemy) ChasePlayer(px, py int) {
	dirX := 0
	dirY := 0

	if e.X < px {
		dirX = 1
	} else if e.X > px {
		dirX = -1
	}

	if e.Y < py {
		dirY = 1
	} else if e.Y > py {
		dirY = -1
	}

	// Погрешность 15% — враг иногда "ошибается"
	if rand.Intn(100) < 15 {
		if dirX != 0 && rand.Intn(2) == 0 {
			dirX = -dirX
		}
		if dirY != 0 && rand.Intn(2) == 0 {
			dirY = -dirY
		}
	}

	nx := e.X + dirX
	ny := e.Y + dirY

	if nx >= 1 && nx <= gameWidth-2 {
		e.X = nx
	}
	if ny >= 1 && ny <= gameHeight-2 {
		e.Y = ny
	}
}

func (e *Enemy) Draw(screen tcell.Screen) {
	screen.SetContent(offsetX+e.X, offsetY+e.Y, e.Char, nil, e.Style)
}

// ========================================================================
// ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ
// ========================================================================

func drawString(screen tcell.Screen, x, y int, msg string, style tcell.Style) {
	for i, char := range msg {
		screen.SetContent(x+i, y, char, nil, style)
	}
}

func drawBorder(screen tcell.Screen) {
	style := tcell.StyleDefault.Foreground(colorBorder)
	for x := 1; x < gameWidth-1; x++ {
		screen.SetContent(offsetX+x, offsetY, borderH, nil, style)
		screen.SetContent(offsetX+x, offsetY+gameHeight-1, borderH, nil, style)
	}
	for y := 1; y < gameHeight-1; y++ {
		screen.SetContent(offsetX, offsetY+y, borderV, nil, style)
		screen.SetContent(offsetX+gameWidth-1, offsetY+y, borderV, nil, style)
	}
	screen.SetContent(offsetX, offsetY, cornerTopL, nil, style)
	screen.SetContent(offsetX+gameWidth-1, offsetY, cornerTopR, nil, style)
	screen.SetContent(offsetX, offsetY+gameHeight-1, cornerBotL, nil, style)
	screen.SetContent(offsetX+gameWidth-1, offsetY+gameHeight-1, cornerBotR, nil, style)
}

// abs — модуль числа.
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ========================================================================
// ГЕНЕРАЦИЯ ОБЪЕКТОВ С ЗАЩИТОЙ ОТ НАЛОЖЕНИЯ
// ========================================================================

// Генерация монет с разными типами
func setupCoins(level int, playerX, playerY int, enemies []*Enemy, powerUps []*PowerUp) []*Coin {
	coins := make([]*Coin, 0, level+2)
	occupied := make(map[[2]int]bool)
	occupied[[2]int{playerX, playerY}] = true
	for _, e := range enemies {
		occupied[[2]int{e.X, e.Y}] = true
	}
	for _, p := range powerUps {
		occupied[[2]int{p.X, p.Y}] = true
	}

	targetCount := level + 2
	attempts := 0
	maxAttempts := 1000

	for len(coins) < targetCount && attempts < maxAttempts {
		attempts++
		x := rand.Intn(gameWidth-2) + 1
		y := rand.Intn(gameHeight-2) + 1
		if occupied[[2]int{x, y}] {
			continue
		}
		occupied[[2]int{x, y}] = true
		dx := rand.Intn(3) - 1
		dy := rand.Intn(3) - 1
		if dx == 0 && dy == 0 {
			dx = 1
		}
		coinType := randomCoinType()
		coins = append(coins, NewCoin(coinTypeChars[coinType], x, y, dx, dy, coinType))
	}
	return coins
}

// Генерация врагов с умным преследователем (с уровня 2)
func setupEnemies(level int, playerX, playerY int, coins []*Coin, powerUps []*PowerUp) []*Enemy {
	enemies := make([]*Enemy, 0, level+1)
	occupied := make(map[[2]int]bool)
	occupied[[2]int{playerX, playerY}] = true
	for _, c := range coins {
		occupied[[2]int{c.X, c.Y}] = true
	}
	for _, p := range powerUps {
		occupied[[2]int{p.X, p.Y}] = true
	}

	// Обычные враги
	targetCount := level
	attempts := 0
	maxAttempts := 1000

	for len(enemies) < targetCount && attempts < maxAttempts {
		attempts++
		x := rand.Intn(gameWidth-2) + 1
		y := rand.Intn(gameHeight-2) + 1
		if occupied[[2]int{x, y}] {
			continue
		}
		occupied[[2]int{x, y}] = true
		dx := rand.Intn(3) - 1
		dy := rand.Intn(3) - 1
		if dx == 0 && dy == 0 {
			dx = 1
		}
		enemies = append(enemies, NewEnemy('X', x, y, dx, dy, false))
	}

	// Умный враг-преследователь (с уровня 2)
	if level >= 2 {
		attempts = 0
		for attempts < maxAttempts {
			attempts++
			x := rand.Intn(gameWidth-2) + 1
			y := rand.Intn(gameHeight-2) + 1
			if occupied[[2]int{x, y}] {
				continue
			}
			// Умный враг появляется минимум в 10 клетках от игрока
			dist := abs(x-playerX) + abs(y-playerY)
			if dist < 10 {
				continue
			}
			occupied[[2]int{x, y}] = true
			enemies = append(enemies, NewEnemy('Z', x, y, 0, 0, true))
			break
		}
	}
	return enemies
}

// Генерация бонусов (сердечки с вероятностью 30%)
func setupPowerUps(level int, playerX, playerY int, coins []*Coin, enemies []*Enemy) []*PowerUp {
	powerUps := make([]*PowerUp, 0)

	if rand.Intn(100) >= 30 {
		return powerUps
	}

	occupied := make(map[[2]int]bool)
	occupied[[2]int{playerX, playerY}] = true
	for _, c := range coins {
		occupied[[2]int{c.X, c.Y}] = true
	}
	for _, e := range enemies {
		occupied[[2]int{e.X, e.Y}] = true
	}

	attempts := 0
	maxAttempts := 1000
	for attempts < maxAttempts {
		attempts++
		x := rand.Intn(gameWidth-2) + 1
		y := rand.Intn(gameHeight-2) + 1
		if occupied[[2]int{x, y}] {
			continue
		}
		powerUps = append(powerUps, &PowerUp{
			X:    x,
			Y:    y,
			Type: PowerUpHeart,
		})
		break
	}
	return powerUps
}

// ========================================================================
// ФУНКЦИЯ СОЗДАНИЯ ЧАСТИЦ
// ========================================================================
func spawnParticles(particles *[]*Particle, x, y, points int, color tcell.Color) {
	var text string
	if points > 0 {
		text = fmt.Sprintf("+%d", points)
	} else {
		text = fmt.Sprintf("%d", points)
	}

	style := tcell.StyleDefault.Foreground(color).Bold(true)

	for i, ch := range text {
		*particles = append(*particles, &Particle{
			X:     x - len(text)/2 + i,
			Y:     y,
			Char:  ch,
			Style: style,
			DirY:  -1,
		})
	}
}

// ========================================================================
// ГЛАВНАЯ ФУНКЦИЯ
// ========================================================================
func main() {
	// === ЛОГИРОВАНИЕ ===
	logFile, err := os.Create("game.log")
	if err != nil {
		fmt.Println("Не удалось создать game.log:", err)
	} else {
		defer logFile.Close()
		log.SetOutput(logFile)
		log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
		log.Println("=== Запуск игры ===")
	}

	// === ЭКРАН ===
	screen, err := tcell.NewScreen()
	if err != nil {
		log.Fatal(err)
	}
	defer screen.Fini()
	if err := screen.Init(); err != nil {
		log.Fatal(err)
	}

	// === ЗВУК ===
	sound := NewSoundManager("sounds")
	defer sound.Close()
	sound.PlayMusic("sounds/music.mp3")

	// === ИГРОВЫЕ ПЕРЕМЕННЫЕ ===
	gameState := StateStart
	player := NewSprite('@', gameWidth/2, gameHeight/2)
	player.Style = tcell.StyleDefault.Foreground(colorPlayer)

	health := 3
	score := 0
	level := 1

	// Генерация объектов на 1-м уровне
	enemies := setupEnemies(1, player.X, player.Y, nil, nil)
	coins := setupCoins(1, player.X, player.Y, enemies, nil)
	powerUps := setupPowerUps(1, player.X, player.Y, coins, enemies)
	particles := make([]*Particle, 0)

	coinsMoving := true
	invulnerableUntil := time.Now()

	// === ТАЙМЕРЫ ===
	lastCoinAnim := time.Now()
	coinAnimDelay := 150 * time.Millisecond
	lastCoinMove := time.Now()
	coinMoveDelay := 300 * time.Millisecond
	lastEnemyMove := time.Now()
	enemyMoveDelay := 250 * time.Millisecond
	lastParticleUpdate := time.Now()
	particleUpdateDelay := 100 * time.Millisecond
	animFrame := 0

	// === ГЛАВНЫЙ ИГРОВОЙ ЦИКЛ ===
	running := true
	for running {

		// ============================================================
		// СТАРТОВЫЙ ЭКРАН
		// ============================================================
		if gameState == StateStart {
			screen.Clear()
			drawString(screen, 10, 6, "=== TERMINAL GAME ===", tcell.StyleDefault.Foreground(colorUI))
			drawString(screen, 8, 8, "WASD/Стрелки: движение", tcell.StyleDefault.Foreground(colorUI))
			drawString(screen, 8, 9, "E: движение монет", tcell.StyleDefault.Foreground(colorUI))
			drawString(screen, 8, 10, "M: музыка, N: звуки", tcell.StyleDefault.Foreground(colorUI))
			drawString(screen, 8, 11, "-/=: громкость (0.5-8.0)", tcell.StyleDefault.Foreground(colorUI))
			drawString(screen, 8, 13, "Собирайте: 0(1) $(5) ♦(10) ♥(+жизнь)", tcell.StyleDefault.Foreground(colorUI))
			drawString(screen, 8, 14, "Избегайте: X(враги) Z(преследователь)", tcell.StyleDefault.Foreground(colorUI))
			drawString(screen, 8, 16, "Нажмите любую клавишу для старта", tcell.StyleDefault.Foreground(colorUI))
			screen.Show()

			for {
				if screen.HasPendingEvent() {
					ev := screen.PollEvent()
					if key, ok := ev.(*tcell.EventKey); ok {
						if key.Rune() == 'q' || key.Key() == tcell.KeyEscape {
							running = false
						} else {
							gameState = StatePlaying
						}
						break
					}
				}
				time.Sleep(16 * time.Millisecond)
				if !running {
					break
				}
			}
			continue
		}

		// ============================================================
		// ЭКРАН GAME OVER
		// ============================================================
		if gameState == StateGameOver {
			screen.Clear()
			drawString(screen, 12, 8, "=== GAME OVER ===", tcell.StyleDefault.Foreground(colorEnemy))
			drawString(screen, 12, 10, fmt.Sprintf("Счёт: %d", score), tcell.StyleDefault.Foreground(colorEnemy))
			drawString(screen, 12, 11, fmt.Sprintf("Уровень: %d", level), tcell.StyleDefault.Foreground(colorEnemy))
			drawString(screen, 10, 13, "Нажмите Q для выхода", tcell.StyleDefault.Foreground(colorEnemy))
			screen.Show()

			for {
				if screen.HasPendingEvent() {
					ev := screen.PollEvent()
					if key, ok := ev.(*tcell.EventKey); ok {
						if key.Rune() == 'q' {
							running = false
						}
						break
					}
				}
				time.Sleep(16 * time.Millisecond)
				if !running {
					break
				}
			}
			continue
		}

		// ============================================================
		// ОБНОВЛЕНИЕ ТАЙМЕРОВ
		// ============================================================
		now := time.Now()

		// Анимация переливания монет
		if now.Sub(lastCoinAnim) > coinAnimDelay {
			animFrame++
			lastCoinAnim = now
		}

		// Движение монет
		if coinsMoving && now.Sub(lastCoinMove) > coinMoveDelay {
			for _, c := range coins {
				c.Update()
			}
			lastCoinMove = now

			// Автоматический сбор монет (когда монета прилетает на игрока)
			if gameState == StatePlaying {
				for i := len(coins) - 1; i >= 0; i-- {
					c := coins[i]
					if c.X == player.X && c.Y == player.Y {
						points := c.Type.Points()
						score += points
						sound.PlayCoin()
						spawnParticles(&particles, c.X, c.Y, points, coinTypeBaseColors[c.Type])

						coins[i] = coins[len(coins)-1]
						coins = coins[:len(coins)-1]

						if len(coins) == 0 {
							level++
							sound.PlayLevelUp()
							enemies = setupEnemies(level, player.X, player.Y, nil, nil)
							coins = setupCoins(level, player.X, player.Y, enemies, nil)
							powerUps = setupPowerUps(level, player.X, player.Y, coins, enemies)

							newCoinDelay := 300*time.Millisecond - time.Duration(level*20)*time.Millisecond
							if newCoinDelay < 100*time.Millisecond {
								newCoinDelay = 100 * time.Millisecond
							}
							coinMoveDelay = newCoinDelay

							newEnemyDelay := 250*time.Millisecond - time.Duration(level*15)*time.Millisecond
							if newEnemyDelay < 80*time.Millisecond {
								newEnemyDelay = 80 * time.Millisecond
							}
							enemyMoveDelay = newEnemyDelay
						}
						break
					}
				}

				// Сбор бонусов
				for i := len(powerUps) - 1; i >= 0; i-- {
					p := powerUps[i]
					if p.X == player.X && p.Y == player.Y {
						if p.Type == PowerUpHeart {
							if health < maxHealth {
								health++
								spawnParticles(&particles, p.X, p.Y, 1, colorHealth)
							}
						}
						sound.PlayCoin()
						powerUps[i] = powerUps[len(powerUps)-1]
						powerUps = powerUps[:len(powerUps)-1]
					}
				}
			}
		}

		// Движение врагов
		if now.Sub(lastEnemyMove) > enemyMoveDelay {
			for _, e := range enemies {
				e.Update(player.X, player.Y)

				// Проверка неуязвимости
				if e.X == player.X && e.Y == player.Y && now.Before(invulnerableUntil) {
					continue
				}

				if e.X == player.X && e.Y == player.Y {
					health--
					sound.PlayHit()
					spawnParticles(&particles, player.X, player.Y, -1, colorEnemy)
					if health <= 0 {
						gameState = StateGameOver
					} else {
						player.X = gameWidth / 2
						player.Y = gameHeight / 2
						invulnerableUntil = now.Add(invulnerableDuration)
					}
				}
			}
			lastEnemyMove = now
		}

		// Обновление частиц
		if now.Sub(lastParticleUpdate) > particleUpdateDelay {
			for i := len(particles) - 1; i >= 0; i-- {
				particles[i].Y += particles[i].DirY
				if particles[i].Y < 1 || particles[i].Y > gameHeight-2 {
					particles[i] = particles[len(particles)-1]
					particles = particles[:len(particles)-1]
				}
			}
			lastParticleUpdate = now
		}

		// ============================================================
		// ОТРИСОВКА КАДРА
		// ============================================================
		screen.Clear()
		termW, termH := screen.Size()
		if termW < gameWidth || termH < gameHeight+offsetY {
			drawString(screen, 1, 1, "Увеличьте окно терминала!", tcell.StyleDefault.Foreground(colorEnemy))
			screen.Show()
			time.Sleep(100 * time.Millisecond)
			continue
		}

		drawBorder(screen)

		// Игрок (с миганием при неуязвимости)
		isInvulnerable := now.Before(invulnerableUntil)
		playerVisible := true
		if isInvulnerable {
			playerVisible = (animFrame % 2) == 0
		}
		if playerVisible {
			screen.SetContent(offsetX+player.X, offsetY+player.Y, player.Char, nil, player.Style)
		}

		// Монеты с уникальной анимацией переливания для каждого типа
		cf := animFrame % 6
		for _, c := range coins {
			var char rune
			var color tcell.Color
			switch c.Type {
			case CoinNormal:
				char = coinNormalFrames[cf]
				color = coinNormalColors[cf]
			case CoinGold:
				char = coinGoldFrames[cf]
				color = coinGoldColors[cf]
			case CoinRare:
				char = coinRareFrames[cf]
				color = coinRareColors[cf]
			default:
				char = coinNormalFrames[cf]
				color = coinNormalColors[cf]
			}
			screen.SetContent(
				offsetX+c.X,
				offsetY+c.Y,
				char,
				nil,
				tcell.StyleDefault.Foreground(color),
			)
		}

		// Враги
		for _, e := range enemies {
			e.Draw(screen)
		}

		// Бонусы (сердечки) с пульсацией
		for _, p := range powerUps {
			style := tcell.StyleDefault.Foreground(p.Type.Color())
			if animFrame%4 < 2 {
				style = style.Bold(true)
			}
			screen.SetContent(offsetX+p.X, offsetY+p.Y, p.Type.Char(), nil, style)
		}

		// Частицы
		for _, p := range particles {
			if p.Y >= 1 && p.Y <= gameHeight-2 {
				screen.SetContent(offsetX+p.X, offsetY+p.Y, p.Char, nil, p.Style)
			}
		}

		// ============================================================
		// UI
		// ============================================================
		drawString(screen, 1, 0, "=== Terminal Game ===", tcell.StyleDefault.Foreground(colorUI))
		drawString(screen, 1, 1, fmt.Sprintf("Score: %d", score), tcell.StyleDefault.Foreground(colorUI))
		drawString(screen, 1, 2, fmt.Sprintf("Level: %d", level), tcell.StyleDefault.Foreground(colorUI))

		// Полоска здоровья
		hp := "HP: "
		for i := 0; i < health; i++ {
			hp += "♥"
		}
		for i := health; i < maxHealth; i++ {
			hp += "♡"
		}
		drawString(screen, 20, 1, hp, tcell.StyleDefault.Foreground(colorHealth))

		// Индикатор движения монет
		cStatus := "Coins: ON "
		cStyle := tcell.ColorGreen
		if !coinsMoving {
			cStatus = "Coins: OFF"
			cStyle = tcell.ColorRed
		}
		drawString(screen, 20, 2, cStatus, tcell.StyleDefault.Foreground(cStyle))

		// Индикатор громкости (совместимо с новой версией sound.go)
		if sound.IsMusicOn() {
			volStr := fmt.Sprintf("♪ %.1f", sound.GetDisplayVolume())
			drawString(screen, 32, 2, volStr, tcell.StyleDefault.Foreground(tcell.ColorYellow))
		} else {
			drawString(screen, 32, 2, "♪ OFF", tcell.StyleDefault.Foreground(tcell.ColorGray))
		}

		screen.Show()

		// ============================================================
		// ОБРАБОТКА ВВОДА
		// ============================================================
		playerMoved := false
		for screen.HasPendingEvent() {
			ev := screen.PollEvent()
			if key, ok := ev.(*tcell.EventKey); ok {
				// Пауза
				if key.Key() == tcell.KeyEscape && gameState == StatePlaying {
					gameState = StatePaused
					drawString(screen, 15, 10, "ПАУЗА (Esc)", tcell.StyleDefault.Foreground(colorUI))
					drawString(screen, 8, 12, "-/=: громкость, M/N: вкл/выкл", tcell.StyleDefault.Foreground(colorUI))
					screen.Show()

					for {
						pe := screen.PollEvent()
						if pauseKey, ok := pe.(*tcell.EventKey); ok {
							switch pauseKey.Key() {
							case tcell.KeyEscape:
								gameState = StatePlaying
							}
							switch pauseKey.Rune() {
							case 'm', 'M':
								sound.ToggleMusic()
							case 'n', 'N':
								sound.ToggleSound()
							case '-', '_':
								sound.VolumeDown()
							case '=', '+':
								sound.VolumeUp()
							}
							if gameState == StatePlaying {
								break
							}
						}
						time.Sleep(16 * time.Millisecond)
					}
					continue
				}

				if gameState == StatePlaying {
					switch key.Key() {
					case tcell.KeyUp:
						if player.Y > 1 {
							player.Y--
							playerMoved = true
						}
					case tcell.KeyDown:
						if player.Y < gameHeight-2 {
							player.Y++
							playerMoved = true
						}
					case tcell.KeyLeft:
						if player.X > 1 {
							player.X--
							playerMoved = true
						}
					case tcell.KeyRight:
						if player.X < gameWidth-2 {
							player.X++
							playerMoved = true
						}
					}

					switch key.Rune() {
					case 'q':
						running = false
					case 'w':
						if player.Y > 1 {
							player.Y--
							playerMoved = true
						}
					case 's':
						if player.Y < gameHeight-2 {
							player.Y++
							playerMoved = true
						}
					case 'a':
						if player.X > 1 {
							player.X--
							playerMoved = true
						}
					case 'd':
						if player.X < gameWidth-2 {
							player.X++
							playerMoved = true
						}
					case 'e', 'E':
						coinsMoving = !coinsMoving
					case 'm', 'M':
						sound.ToggleMusic()
					case 'n', 'N':
						sound.ToggleSound()
					case '-', '_':
						sound.VolumeDown()
					case '=', '+':
						sound.VolumeUp()
					}
				}
			} else if _, ok := ev.(*tcell.EventResize); ok {
				screen.Sync()
			}
		}

		// ============================================================
		// ПРОВЕРКА СТОЛКНОВЕНИЙ (ПРИ ДВИЖЕНИИ ИГРОКА)
		// ============================================================
		if playerMoved && gameState == StatePlaying {
			// Монеты
			for i := len(coins) - 1; i >= 0; i-- {
				c := coins[i]
				if c.X == player.X && c.Y == player.Y {
					points := c.Type.Points()
					score += points
					sound.PlayCoin()
					spawnParticles(&particles, c.X, c.Y, points, coinTypeBaseColors[c.Type])

					coins[i] = coins[len(coins)-1]
					coins = coins[:len(coins)-1]

					if len(coins) == 0 {
						level++
						sound.PlayLevelUp()
						enemies = setupEnemies(level, player.X, player.Y, nil, nil)
						coins = setupCoins(level, player.X, player.Y, enemies, nil)
						powerUps = setupPowerUps(level, player.X, player.Y, coins, enemies)

						newCoinDelay := 300*time.Millisecond - time.Duration(level*20)*time.Millisecond
						if newCoinDelay < 100*time.Millisecond {
							newCoinDelay = 100 * time.Millisecond
						}
						coinMoveDelay = newCoinDelay

						newEnemyDelay := 250*time.Millisecond - time.Duration(level*15)*time.Millisecond
						if newEnemyDelay < 80*time.Millisecond {
							newEnemyDelay = 80 * time.Millisecond
						}
						enemyMoveDelay = newEnemyDelay
					}
					break
				}
			}

			// Бонусы
			for i := len(powerUps) - 1; i >= 0; i-- {
				p := powerUps[i]
				if p.X == player.X && p.Y == player.Y {
					if p.Type == PowerUpHeart {
						if health < maxHealth {
							health++
							spawnParticles(&particles, p.X, p.Y, 1, colorHealth)
						}
					}
					sound.PlayCoin()
					powerUps[i] = powerUps[len(powerUps)-1]
					powerUps = powerUps[:len(powerUps)-1]
				}
			}
		}

		time.Sleep(16 * time.Millisecond)
	}
}