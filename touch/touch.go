// touch command modeled on heirloom project's touch
package main

import (
	"syscall"
	"flag"
	"fmt"
	"os"
	"strings"
	"strconv"
	"time"
)

var aflag = flag.Bool("a", false, "Change the access time")
var mflag = flag.Bool("m", false, "Change the modification time")
var cflag = flag.Bool("c", false, "Don't create the file if not exists")
var tflag = flag.String("t", "", "Use time instead of current time. Time specified as [[CC]YY]MMDDhhmm[.SS]")
var rflag = flag.String("r", "", "ref_file Use time of corresponding file rather than current time")

var now = time.Seconds()

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: touch [-amc] [-r ref_file] [-t time] file ...")
	os.Exit(2)
}

func badtime() {
	fmt.Fprintf(os.Stderr, "touch: bad time specification\n")
	os.Exit(2)
}

func atot(s string, m int) int {
	i, e := strconv.Atoi(s)

	if i > m || i < 0 || s[0] == '+' || s[0] == '-' {
		badtime()
	}

	return i
}

func ptime(cp string) syscall.Time_t {
	t := now
	stm := time.LocalTime()

	if sz := len(cp); sz == 11 || sz == 13 || sz == 15 {
		if strings.Index(cp, ".") != sz-3 {
			badtime()
		}

		stm.Second = atot(cp[sz-2:], 61)
		sz -= 3
	} else {
		stm.Second = 0
	}

	if sz == 12 {
		year := cp[:4]
		if stm.Year = atot(year, 30000); stm.Year < 1979 {
			badtime()
		}
	}

}

func main() {
	flag.Parse()

	if flag.NArg() == 0 {
		usage()
	}

	t := flag.Lookup("t")
	if t != nil {
		if t.Value == nil {
			usage()
		}
	}
}
