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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/mem/labyrinth/mazelib"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Defining the icarus command.
// This will be called as 'laybrinth icarus'
var icarusCmd = &cobra.Command{
	Use:     "icarus",
	Aliases: []string{"client"},
	Short:   "Start the laybrinth solver",
	Long: `Icarus wakes up to find himself in the middle of a Labyrinth.
  Due to the darkness of the Labyrinth he can only see his immediate cell and if
  there is a wall or not to the top, right, bottom and left. He takes one step
  and then can discover if his new cell has walls on each of the four sides.

  Icarus can connect to a Daedalus and solve many laybrinths at a time.`,
	Run: func(cmd *cobra.Command, args []string) {
		RunIcarus()
	},
}

func init() {
	RootCmd.AddCommand(icarusCmd)
}

func RunIcarus() {
	// Run the solver as many times as the user desires.
	fmt.Println("Solving", viper.GetInt("times"), "times")
	for x := 0; x < viper.GetInt("times"); x++ {

		solveMaze()
	}

	// Once we have solved the maze the required times, tell daedalus we are done
	makeRequest("http://127.0.0.1:" + viper.GetString("port") + "/done")
}

// Make a call to the laybrinth server (daedalus) that icarus is ready to wake up
func awake() mazelib.Survey {
	contents, err := makeRequest("http://127.0.0.1:" + viper.GetString("port") + "/awake")
	if err != nil {
		fmt.Println(err)
	}
	r := ToReply(contents)
	return r.Survey
}

// Make a call to the laybrinth server (daedalus)
// to move Icarus a given direction
// Will be used heavily by solveMaze
func Move(direction string) (mazelib.Survey, error) {
	if direction == "left" || direction == "right" || direction == "up" || direction == "down" {

		contents, err := makeRequest("http://127.0.0.1:" + viper.GetString("port") + "/move/" + direction)
		if err != nil {
			return mazelib.Survey{}, err
		}

		rep := ToReply(contents)
		switch {
		case rep.Victory == true:
			fmt.Println(rep.Message)
			// os.Exit(1)
			return rep.Survey, mazelib.ErrVictory
		case rep.Error == true:
			return rep.Survey, errors.New(rep.Message)
		default:
			return rep.Survey, nil
		}
	}

	return mazelib.Survey{}, errors.New("invalid direction")
}

// UndoMove tells the server to move Icarus in the opposite direction of
// `direction` to support backing off.
func UndoMove(direction string) (mazelib.Survey, error) {
	switch direction {
	case "left":
		direction = "right"
	case "right":
		direction = "left"
	case "up":
		direction = "down"
	case "down":
		direction = "up"
	}
	return Move(direction)
}

// utility function to wrap making requests to the daedalus server
func makeRequest(url string) ([]byte, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	return contents, nil
}

// Handling a JSON response and unmarshalling it into a reply struct
func ToReply(in []byte) mazelib.Reply {
	res := &mazelib.Reply{}
	json.Unmarshal(in, &res)
	return *res
}

func solveMaze() {
	s := awake() // Need to start with waking up to initialize a new maze

	RecursiveSolve(s)
}

type pos struct {
	x, y int
}

// RecursiveSolver solves mazes recursively, implementing a backtracking
// strategy
type RecursiveSolver struct {
	visited map[pos]bool
}

// RecursiveSolve creates a RecursiveSolver and solves the maze,
// returning true if a solution was found
func RecursiveSolve(s mazelib.Survey) bool {
	return NewRecursiveSolver().solve(s, pos{})
}

// NewRecursiveSolver creates a new recursive solver
func NewRecursiveSolver() *RecursiveSolver {
	return &RecursiveSolver{
		visited: make(map[pos]bool),
	}
}

// solve solves the maze by keeping track of the visited rooms and
// exploring all the reachable rooms until a solution is found or no
// more options are available. It has a right and bottom bias.
func (solver *RecursiveSolver) solve(s mazelib.Survey, p pos) bool {
	solver.visited[p] = true
	if s.Right == false && solver.right(p) {
		return true
	}
	if s.Bottom == false && solver.down(p) {
		return true
	}
	if s.Left == false && solver.left(p) {
		return true
	}
	if s.Top == false && solver.up(p) {
		return true
	}
	return false
}

// right moves Icarus east
func (solver *RecursiveSolver) right(p pos) bool {
	x, y := mazelib.Shift(p.x, p.y, mazelib.E)
	return solver.visit("right", pos{x, y})
}

// down moves Icarus south
func (solver *RecursiveSolver) down(p pos) bool {
	x, y := mazelib.Shift(p.x, p.y, mazelib.S)
	return solver.visit("down", pos{x, y})
}

// left moves Icarus west
func (solver *RecursiveSolver) left(p pos) bool {
	x, y := mazelib.Shift(p.x, p.y, mazelib.W)
	return solver.visit("left", pos{x, y})
}

// up moves Icarus north
func (solver *RecursiveSolver) up(p pos) bool {
	x, y := mazelib.Shift(p.x, p.y, mazelib.N)
	return solver.visit("up", pos{x, y})
}

// visit is the common method all the movements use to tell the server
// the direction we want to move, call solve on the new position, and
// back off if that move didn't find a solution.
func (solver *RecursiveSolver) visit(dir string, p pos) bool {
	if solver.visited[p] {
		return false
	}
	switch t, err := Move(dir); err {
	case nil:
		if solver.solve(t, p) {
			return true
		}
		UndoMove(dir)
	case mazelib.ErrVictory:
		return true
	}
	return false
}
