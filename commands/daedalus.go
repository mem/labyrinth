// Copyright © 2015 Steve Francia <spf@spf13.com>.
// Copyright © 2015 Marcelo E. Magallon <marcelo.magallon@gmail.com>.
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.
//

package commands

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mem/labyrinth/mazelib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Maze struct {
	rooms      [][]mazelib.Room
	start      mazelib.Coordinate
	end        mazelib.Coordinate
	icarus     mazelib.Coordinate
	StepsTaken int
}

// Tracking the current maze being solved

// WARNING: This approach is not safe for concurrent use
// This server is only intended to have a single client at a time
// We would need a different and more complex approach if we wanted
// concurrent connections than these simple package variables
var currentMaze *Maze
var scores []int

// Defining the daedalus command.
// This will be called as 'laybrinth daedalus'
var daedalusCmd = &cobra.Command{
	Use:     "daedalus",
	Aliases: []string{"deadalus", "server"},
	Short:   "Start the laybrinth creator",
	Long: `Daedalus's job is to create a challenging Labyrinth for his opponent
  Icarus to solve.

  Daedalus runs a server which Icarus clients can connect to to solve laybrinths.`,
	Run: func(cmd *cobra.Command, args []string) {
		RunServer()
	},
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano()) // need to initialize the seed
	gin.SetMode(gin.ReleaseMode)

	RootCmd.AddCommand(daedalusCmd)
}

// Runs the web server
func RunServer() {
	// Adding handling so that even when ctrl+c is pressed we still print
	// out the results prior to exiting.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		printResults()
		os.Exit(1)
	}()

	// Using gin-gonic/gin to handle our routing
	r := gin.Default()
	v1 := r.Group("/")
	{
		v1.GET("/awake", GetStartingPoint)
		v1.GET("/move/:direction", MoveDirection)
		v1.GET("/done", End)
	}

	r.Run(":" + viper.GetString("port"))
}

// Ends a session and prints the results.
// Called by Icarus when he has reached
//   the number of times he wants to solve the laybrinth.
func End(c *gin.Context) {
	printResults()
	os.Exit(1)
}

// initializes a new maze and places Icarus in his awakening location
func GetStartingPoint(c *gin.Context) {
	initializeMaze()
	startRoom, err := currentMaze.Discover(currentMaze.Icarus())
	if err != nil {
		fmt.Println("Icarus is outside of the maze. This shouldn't ever happen")
		fmt.Println(err)
		os.Exit(-1)
	}
	if viper.GetBool("pretty") {
		mazelib.PrintPrettyMaze(currentMaze)
	} else {
		mazelib.PrintMaze(currentMaze)
	}

	c.JSON(http.StatusOK, mazelib.Reply{Survey: startRoom})
}

// The API response to the /move/:direction address
func MoveDirection(c *gin.Context) {
	var err error

	switch c.Param("direction") {
	case "left":
		err = currentMaze.MoveLeft()
	case "right":
		err = currentMaze.MoveRight()
	case "down":
		err = currentMaze.MoveDown()
	case "up":
		err = currentMaze.MoveUp()
	}

	var r mazelib.Reply

	if err != nil {
		r.Error = true
		r.Message = err.Error()
		c.JSON(409, r)
		return
	}

	s, e := currentMaze.LookAround()

	if e != nil {
		if e == mazelib.ErrVictory {
			scores = append(scores, currentMaze.StepsTaken)
			r.Victory = true
			r.Message = fmt.Sprintf("Victory achieved in %d steps \n", currentMaze.StepsTaken)
		} else {
			r.Error = true
			r.Message = err.Error()
		}
	}

	r.Survey = s

	c.JSON(http.StatusOK, r)
}

func initializeMaze() {
	currentMaze = createMaze()
}

// Print to the terminal the average steps to solution for the current session
func printResults() {
	fmt.Printf("Labyrinth solved %d times with an avg of %d steps\n", len(scores), mazelib.AvgScores(scores))
}

// Return a room from the maze
func (m *Maze) GetRoom(x, y int) (*mazelib.Room, error) {
	if x < 0 || y < 0 || x >= m.Width() || y >= m.Height() {
		return &mazelib.Room{}, errors.New("room outside of maze boundaries")
	}

	return &m.rooms[y][x], nil
}

func (m *Maze) Width() int  { return len(m.rooms[0]) }
func (m *Maze) Height() int { return len(m.rooms) }

// Return Icarus's current position
func (m *Maze) Icarus() (x, y int) {
	return m.icarus.X, m.icarus.Y
}

// Set the location where Icarus will awake
func (m *Maze) SetStartPoint(x, y int) error {
	r, err := m.GetRoom(x, y)

	if err != nil {
		return err
	}

	if r.Treasure {
		return errors.New("can't start in the treasure")
	}

	r.Start = true
	m.icarus = mazelib.Coordinate{x, y}
	return nil
}

// Set the location of the treasure for a given maze
func (m *Maze) SetTreasure(x, y int) error {
	r, err := m.GetRoom(x, y)

	if err != nil {
		return err
	}

	if r.Start {
		return errors.New("can't have the treasure at the start")
	}

	r.Treasure = true
	m.end = mazelib.Coordinate{x, y}
	return nil
}

// Given Icarus's current location, Discover that room
// Will return ErrVictory if Icarus is at the treasure.
func (m *Maze) LookAround() (mazelib.Survey, error) {
	if m.end.X == m.icarus.X && m.end.Y == m.icarus.Y {
		fmt.Printf("Victory achieved in %d steps \n", m.StepsTaken)
		return mazelib.Survey{}, mazelib.ErrVictory
	}

	return m.Discover(m.icarus.X, m.icarus.Y)
}

// Given two points, survey the room.
// Will return error if two points are outside of the maze
func (m *Maze) Discover(x, y int) (mazelib.Survey, error) {
	if r, err := m.GetRoom(x, y); err != nil {
		return mazelib.Survey{}, nil
	} else {
		return r.Walls, nil
	}
}

// Moves Icarus's position left one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveLeft() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Left {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x-1, y); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x - 1, y}
	m.StepsTaken++
	return nil
}

// Moves Icarus's position right one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveRight() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Right {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x+1, y); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x + 1, y}
	m.StepsTaken++
	return nil
}

// Moves Icarus's position up one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveUp() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Top {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x, y-1); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x, y - 1}
	m.StepsTaken++
	return nil
}

// Moves Icarus's position down one step
// Will not permit moving through walls or out of the maze
func (m *Maze) MoveDown() error {
	s, e := m.LookAround()
	if e != nil {
		return e
	}
	if s.Bottom {
		return errors.New("Can't walk through walls")
	}

	x, y := m.Icarus()
	if _, err := m.GetRoom(x, y+1); err != nil {
		return err
	}

	m.icarus = mazelib.Coordinate{x, y + 1}
	m.StepsTaken++
	return nil
}

// Creates a maze without any walls
// Good starting point for additive algorithms
func emptyMaze() *Maze {
	z := Maze{}
	ySize := viper.GetInt("height")
	xSize := viper.GetInt("width")

	z.rooms = make([][]mazelib.Room, ySize)
	for y := 0; y < ySize; y++ {
		z.rooms[y] = make([]mazelib.Room, xSize)
		for x := 0; x < xSize; x++ {
			z.rooms[y][x] = mazelib.Room{}
		}
	}

	return &z
}

// Creates a maze with all walls
// Good starting point for subtractive algorithms
func fullMaze() *Maze {
	z := emptyMaze()
	ySize := viper.GetInt("height")
	xSize := viper.GetInt("width")

	for y := 0; y < ySize; y++ {
		for x := 0; x < xSize; x++ {
			z.rooms[y][x].Walls = mazelib.Survey{true, true, true, true}
		}
	}

	return z
}

type mazeBuilder func() *Maze

var tracker struct {
	scorecard   []int
	lastBuilder int
	builders    []mazeBuilder
}

// pickBuilder will keep tabs on the client's progress with each kind of
// maze and randomly select one with a bias towards the kind of maze
// that the client seems to have the most difficulty with.  All the ugly
// details are kept inside the function.
func pickBuilder() mazeBuilder {
	if tracker.scorecard == nil {
		tracker.builders = []mazeBuilder{
			createEmptyMaze,
			createSimpleMaze,
			createRingMaze,
			createBtreeMaze,
			createTreeMaze,
		}
		tracker.scorecard = make([]int, len(tracker.builders))
	}

	// fmt.Println(tracker.scorecard)

	if len(scores) > 0 {
		tracker.scorecard[tracker.lastBuilder] += scores[len(scores)-1]
	}

	lim := int(math.Sqrt(float64(viper.GetInt("times"))))
	if s := len(tracker.builders) * len(tracker.builders); lim < s {
		lim = s
	}

	if len(scores) < lim {
		tracker.lastBuilder = (tracker.lastBuilder + 1) % len(tracker.builders)
	} else {
		total := 0
		for _, v := range tracker.scorecard {
			total += v
		}
		n := rand.Intn(total)
		total = 0
		for i, v := range tracker.scorecard {
			total += v
			if n < total {
				tracker.lastBuilder = i
				break
			}
		}
	}

	return tracker.builders[tracker.lastBuilder]
}

// createMaze creates a maze ready to be used by the client
func createMaze() *Maze {
	builder := pickBuilder()

	m := builder()
	placeObjects(m)
	return m
}

// placeObjects places Icarus and the treasure in the maze, taking care
// to not put Icarus and the treasure in the same location.
func placeObjects(m *Maze) {
	w, h := m.Width(), m.Height()
	sx, sy := rand.Intn(w), rand.Intn(h)
	m.SetStartPoint(sx, sy)
	tx, ty := rand.Intn(w), rand.Intn(h)
	// Don't put stuff on top of each other
	if tx == sy && ty == sy {
		if tx > 0 {
			tx--
		} else {
			tx++
		}
		if ty > 0 {
			ty--
		} else {
			ty++
		}
	}
	m.SetTreasure(tx, ty)
}

// addExternalWalls adds the external walls to maze m, in order to make
// sure that whatever the output of the maze builders, it conforms to
// the requirements of an enclosed maze.
func addExternalWalls(m *Maze) {
	w, h := m.Width(), m.Height()

	for x := 0; x < w; x++ {
		r, _ := m.GetRoom(x, 0)
		r.AddWall(mazelib.N)
		r, _ = m.GetRoom(x, h-1)
		r.AddWall(mazelib.S)
	}

	for y := 0; y < h; y++ {
		r, _ := m.GetRoom(0, y)
		r.AddWall(mazelib.W)
		r, _ = m.GetRoom(w-1, y)
		r.AddWall(mazelib.E)
	}
}

// createEmptyMaze creates a maze without any walls inside. Wall-hughing
// algorithms have a problem with this.
func createEmptyMaze() *Maze {
	m := emptyMaze()
	addExternalWalls(m)

	return m
}

// createSimpleMaze creates a simple maze, topologically it's a straight
// line. Depending on the relative location of Icarus and the treasure,
// and the bias of the solving algorithm, this might cause it to take
// ~2*N steps where N is the number of rooms in the maze.
func createSimpleMaze() *Maze {
	m := emptyMaze()
	w, h := m.Width(), m.Height()

	for y := 0; y < h-1; y++ {
		s, e := 0, w
		if y%2 == 0 {
			e--
		} else {
			s++
		}

		for x := s; x < e; x++ {
			mazelib.AddWall(m, x, y, mazelib.S)
		}
	}

	addExternalWalls(m)

	return m
}

// createRingMaze creates a maze of concentric rings. Since the rings
// are disconnected, this will give wall-hughing algorithms some grief.
// If the maze is large enough (100 rooms), the builder will sacrifice
// some walls for larger groups of fully connected rooms, which cause
// some implementations of backtracking algorithms to visit many rooms.
func createRingMaze() *Maze {
	m := emptyMaze()
	w, h := m.Width(), m.Height()
	step := 1
	// Give backtracking algorithms a hard time by creating regions
	// with lots of options
	if w*h >= 100 {
		step = 2
	}

	for y := step; y < h/2+1; y += step {
		// build a a ring
		for x := y; x < w-y; x++ {
			// top wall
			mazelib.AddWall(m, x, y, mazelib.N)
			// bottom wall
			mazelib.AddWall(m, x, h-y-1, mazelib.S)
		}

		for j := y; j < h-y; j++ {
			// left wall
			mazelib.AddWall(m, y, j, mazelib.W)
			// right wall
			mazelib.AddWall(m, w-y-1, j, mazelib.E)
		}

		// Put connecting doors on opposing sides of the
		// labyrinth in order to force walking all around to get
		// to the next door.
		if (y/step)%2 == 0 {
			mazelib.RmWall(m, y, y, mazelib.W)
		} else {
			mazelib.RmWall(m, w-y-1, h-y-1, mazelib.E)
		}
	}

	addExternalWalls(m)

	return m
}

// createBtreeMaze creates maze with rooms connected to two other rooms.
// Topologically the resulting maze is a tree, which guarantees that the
// maze is perfect. There's always a large hallway to the west and the
// north since most people have a tendency to prefer positive numbers,
// and these directions require substracting.
func createBtreeMaze() *Maze {
	m := fullMaze()
	w, h := m.Width(), m.Height()

	dirs := make([]int, 0, 2)

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dirs = dirs[:0]

			if y != 0 {
				// we can go north
				dirs = append(dirs, mazelib.N)
			}

			if x != 0 {
				// we can go west
				dirs = append(dirs, mazelib.W)
			}

			if len(dirs) > 0 {
				// pick a random direction to go to out
				// of the valid ones and make a passage
				// to it
				dir := dirs[rand.Intn(len(dirs))]
				mazelib.RmWall(m, x, y, dir)
			}
		}
	}

	addExternalWalls(m)

	return m
}

// createTreeMaze creates maze with rooms connected in such a way that
// topologically the resulting maze is a tree, which guarantees that the
// maze is perfect. The maze is as random as it gets, and backtrackers
// will probably excel here, and wall-hughers will always find a
// solution.
func createTreeMaze() *Maze {
	m := fullMaze()
	w, h := m.Width(), m.Height()

	// keep a list of the next rooms where we will be adding
	// passages, add a random room to start with
	c := []pos{{rand.Intn(w), rand.Intn(h)}}

	// keep track of all the rooms we have already visited while
	// building the maze
	visited := make([][]bool, w)
	for i := 0; i < w; i++ {
		visited[i] = make([]bool, h)
	}

	// take note of the unvisited neighbors for the current room
	neighbors := make([]int, 0, 4)

	for len(c) > 0 {
		// visit the most recently added room
		t := c[len(c)-1]
		// and remove it from the list right now
		c = c[:len(c)-1]
		// remember that we will have visited this room (hello, Doctor)
		visited[t.x][t.y] = true

		neighbors = neighbors[:0]

		// check the neighbors in each direction and add them to
		// the list of unvisited ones if necessary
		for _, dir := range []int{mazelib.N, mazelib.E, mazelib.S, mazelib.W} {
			x, y := mazelib.Shift(t.x, t.y, dir)
			if mazelib.Valid(x, y, w, h) && !visited[x][y] {
				neighbors = append(neighbors, dir)
			}
		}

		if len(neighbors) > 0 {
			// pick a random unvisited neighbor out of the valid ones
			dir := neighbors[rand.Intn(len(neighbors))]

			// make a passage to that neighbor
			mazelib.RmWall(m, t.x, t.y, dir)

			// since we found a neighbor, return this room
			// to the list
			c = append(c, t)

			// append the neighbor room to the list that we
			// will be visiting in the future
			x, y := mazelib.Shift(t.x, t.y, dir)
			// appending ensures that this will be the next
			// visited room
			c = append(c, pos{x, y})
		}
	}

	return m
}
