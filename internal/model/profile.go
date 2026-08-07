package model

import "fmt"

type Profile string

func (p Profile) Display() string {
	return fmt.Sprint(string(p))
}
