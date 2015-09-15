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

// This is a small set of interfaces and utilities designed to help
// with the Go Challenge 6: Daedalus & Icarus

package mazelib

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Coordinate describes a location in the maze
type Coordinate struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Reply from the server to a request
type Reply struct {
	Survey  Survey `json:"survey"`
	Victory bool   `json:"victory"`
	Message string `json:"message"`
	Error   bool   `json:"error"`
}

// Survey Given a location, survey surrounding locations
// True indicates a wall is present.
type Survey struct {
	Top    bool `json:"top"`
	Right  bool `json:"right"`
	Bottom bool `json:"bottom"`
	Left   bool `json:"left"`
}

const (
	N = 1
	S = 2
	E = 3
	W = 4
)

var ErrVictory error = errors.New("Victory")

// Room contains the minimum informaion about a room in the maze.
type Room struct {
	Treasure bool
	Start    bool
	Visited  bool
	Walls    Survey
}

func (r *Room) AddWall(dir int) {
	switch dir {
	case N:
		r.Walls.Top = true
	case S:
		r.Walls.Bottom = true
	case E:
		r.Walls.Right = true
	case W:
		r.Walls.Left = true
	}
}

func (r *Room) RmWall(dir int) {
	switch dir {
	case N:
		r.Walls.Top = false
	case S:
		r.Walls.Bottom = false
	case E:
		r.Walls.Right = false
	case W:
		r.Walls.Left = false
	}
}

// MazeI Interface
type MazeI interface {
	GetRoom(x, y int) (*Room, error)
	Width() int
	Height() int
	SetStartPoint(x, y int) error
	SetTreasure(x, y int) error
	LookAround() (Survey, error)
	Discover(x, y int) (Survey, error)
	Icarus() (x, y int)
	MoveLeft() error
	MoveRight() error
	MoveUp() error
	MoveDown() error
}

func AvgScores(in []int) int {
	if len(in) == 0 {
		return 0
	}

	var total int = 0

	for _, x := range in {
		total += x
	}
	return total / (len(in))
}

// PrintMaze : Function to Print Maze to Console
func PrintMaze(m MazeI) {
	fmt.Println("_" + strings.Repeat("___", m.Width()))
	for y := 0; y < m.Height(); y++ {
		str := ""
		for x := 0; x < m.Width(); x++ {
			if x == 0 {
				str += "|"
			}
			r, err := m.GetRoom(x, y)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
			s, err := m.Discover(x, y)
			if err != nil {
				fmt.Println(err)
				os.Exit(-1)
			}
			if s.Bottom {
				if r.Treasure {
					str += "⏅_"
				} else if r.Start {
					str += "⏂_"
				} else {
					str += "__"
				}
			} else {
				if r.Treasure {
					str += "⏃ "
				} else if r.Start {
					str += "⏀ "
				} else {
					str += "  "
				}
			}

			if s.Right {
				str += "|"
			} else {
				str += "_"
			}

		}
		fmt.Println(str)
	}
}

// PrintPrettyMaze prints the maze in a pretty format, which makes
// debugging build issues so much easier. Courtesy of Kim Eik
// (https://gist.github.com/netbrain/63ad3c3743d5ca5e9869)
func PrintPrettyMaze(m MazeI) {
	out := ""
	str := make([][]string, m.Height()*3)
	for i := 0; i < m.Height(); i++ {
		str[i*3] = make([]string, m.Width()*3)
		str[i*3+1] = make([]string, m.Width()*3)
		str[i*3+2] = make([]string, m.Width()*3)
		for j := 0; j < m.Width(); j++ {
			room, _ := m.GetRoom(j, i)
			str[i*3][j*3] = "▛"
			str[i*3][j*3+1] = " "
			str[i*3][j*3+2] = "▜"
			str[i*3+2][j*3] = "▙"
			str[i*3+2][j*3+1] = " "
			str[i*3+2][j*3+2] = "▟"
			str[i*3+1][j*3] = " "
			str[i*3+1][j*3+2] = " "
			str[i*3+1][j*3+1] = " "

			if room.Walls.Top {
				str[i*3][j*3+1] = "▀"
			}

			if room.Walls.Bottom {
				str[i*3+2][j*3+1] = "▄"
			}

			if room.Walls.Left {
				str[i*3+1][j*3] = "▌"
			}

			if room.Walls.Right {
				str[i*3+1][j*3+2] = "▐"
			}

			if room.Visited {
				str[i*3+1][j*3+1] = "·"
			}

			if room.Treasure {
				str[i*3+1][j*3+1] = "×"
			} else if room.Start {
				str[i*3+1][j*3+1] = "⚑"
			}

			x, y := m.Icarus()
			if x == j && y == i {
				str[i*3+1][j*3+1] = "☉"
			}

		}
	}

	for x := 0; x < len(str); x++ {
		for y := 0; y < len(str[x]); y++ {
			out += str[x][y]
		}
		out += "\n"
	}

	fmt.Println(out)
}

// Delta returns the required displacement in order to go in direction
// dir
func Delta(dir int) (dx, dy int) {
	switch dir {
	case E:
		dx = 1
	case W:
		dx = -1
	case N:
		dy = -1
	case S:
		dy = 1
	}

	return dx, dy
}

// Shift takes input coordinates (x, y) and returns displaced
// coordinates in direction dir
func Shift(x, y, dir int) (int, int) {
	dx, dy := Delta(dir)
	return x + dx, y + dy
}

// Reverse returns the reverse direction of dir
func Reverse(dir int) (r int) {
	switch dir {
	case E:
		r = W
	case W:
		r = E
	case N:
		r = S
	case S:
		r = N
	}

	return r
}

// Valid returns true if the positon (x, y) is valid for a maze of size w×h
func Valid(x, y, w, h int) bool {
	return x >= 0 && x < w && y >= 0 && y < h
}

// RmWall removes wall in direction dir from room at position (x, y) in
// maze m. It makes it easier to deal with the fact that all internal
// walls must be double-sized.
func RmWall(m MazeI, x, y, dir int) {
	room, _ := m.GetRoom(x, y)
	room.RmWall(dir)

	x, y = Shift(x, y, dir)
	room, _ = m.GetRoom(x, y)
	room.RmWall(Reverse(dir))
}

// AddWall adds wall in direction dir in room at position (x, y) in maze
// m. It makes it easier to deal with the fact that all internal walls
// must be double-sized.
func AddWall(m MazeI, x, y, dir int) {
	room, _ := m.GetRoom(x, y)
	room.AddWall(dir)

	x, y = Shift(x, y, dir)
	room, _ = m.GetRoom(x, y)
	room.AddWall(Reverse(dir))
}
