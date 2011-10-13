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

var touchFlagset = flag.NewFlagSet("touch",flag.ExitOnError)

var (
	aflag = touchFlagset.Bool("a", false, "Change the access time")
	mflag = touchFlagset.Bool("m", false, "Change the modification time")
	cflag = touchFlagset.Bool("c", false, "Don't create the file if not exists")
	tflag = touchFlagset.String("t", "time", "Use time instead of current time. Time specified as [[CC]YY]MMDDhhmm[.SS]")
	rflag = touchFlagset.String("r", "ref_file", "Use time of corresponding file rather than current time")
)

var (
	nulltime = false // no time given in the arguments
	datetime = true  // Use date_time operand not -t
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: touch [-amc] [-r ref_file] [-t time] file ...")
	os.Exit(2)
}

func badtime() {
	fmt.Fprintf(os.Stderr, "touch: bad time specification\n")
	os.Exit(2)
}

func atot(s string, m int) int {
	i, _ := strconv.Atoi(s)

	if i > m || i < 0 || s[0] == '+' || s[0] == '-' {
		badtime()
	}

	return i
}

func otime(cp string) int64 {
	stm := time.LocalTime()
	datetime = true

	stm.Second = 0

	sz := len(cp)
	switch {
	case sz == 10:
		if stm.Year = int64(atot(cp[8:], 99)); stm.Year < 69 {
			stm.Year += 2000
		} else {
			stm.Year += 1900
		}

		cp = cp[:8]
		sz -= 2; fallthrough
	case sz == 8:
		stm.Minute = atot(cp[6:], 59)
		cp = cp[:6]
		fmt.Println(stm.Minute)

		stm.Hour = atot(cp[4:], 23)
		cp = cp[:4]
		fmt.Println(stm.Hour)

		if stm.Day = atot(cp[2:], 31); stm.Day == 0 {
			badtime()
		}
		fmt.Println(stm.Day)

		cp = cp[:2]

		if stm.Month = atot(cp, 12); stm.Month == 0 {
			badtime()
		}
		stm.Month -= 1
		fmt.Println(stm.Month)
	default:
		badtime()
	}

	return stm.Seconds()
}

func ptime(cp string) int64 {
	stm := time.LocalTime()
	sz := len(cp)

	switch {
	case sz == 11 || sz == 13 || sz == 15:
		if strings.LastIndex(cp, ".") != sz-3 {
			badtime()
		}

		stm.Second = atot(cp[sz-2:], 61)
		cp = cp[:sz-3]
		sz -= 3; fallthrough
	case sz == 12:
		year := cp[:4]
		if stm.Year = int64(atot(year, 30000)); stm.Year < 1970 {
			badtime()
		}

		cp = cp[4:]
		sz -= 4; fallthrough
	case sz == 10:
		var year string
		if tmp := atot(cp[:2], 99); tmp > 69 && tmp <= 99 {
			year = "19" + cp[:2]
		} else {
			year = string(2000 + int(cp[0])*10 + int(cp[1]))
		}

		stm.Year = int64(atot(year, 30000))

		cp = cp[2:]
		sz -= 2; fallthrough
	case sz != 8:
		badtime()
	default:
		stm.Minute = atot(cp[6:], 59)
		cp = cp[:6]

		stm.Hour = atot(cp[4:], 23)
		cp = cp[:4]

		if stm.Day = atot(cp[2:], 31); stm.Day == 0 {
			badtime()
		}
		cp = cp[:2]

		if stm.Month = atot(cp, 12); stm.Month == 0 {
			badtime()
		}
	}

	return stm.Seconds()
}

func reffile(filename string) (int64, int64) {
	var st syscall.Stat_t

	if e := syscall.Stat(filename, &st); e != 0 {
		fmt.Fprintf(os.Stderr, "stat: Error %s\n", syscall.Errstr(e))
		os.Exit(1)
	}

	//FIXME: bug in Go! Amd64 has following rest have Atimespec and Mtimespec
	return st.Atim.Sec, st.Mtim.Sec
}

func touch(filename string, nacc, nmod int64) (errcnt int) {
	var st syscall.Stat_t
	var ut syscall.Utimbuf

	if e := syscall.Stat(filename, &st); e != 0 {
		if e == syscall.ENOENT {

			if *cflag {
				errcnt++
				return
			}

			var fd int
			defer syscall.Close(fd)

			if fd, e = syscall.Creat(filename, 0666); e != 0 {
				fmt.Fprintf(os.Stderr, "touch: can not create %s\n", filename)
				errcnt += 1
				return
			}

			if e = syscall.Fstat(fd, &st); e != 0 {
				fmt.Fprintf(os.Stderr, "touch: can't stat %s\n", filename)
				errcnt += 1
				return
			}
		} else {
			fmt.Fprintf(os.Stderr, "touch: can't stat %s\n", filename)
			errcnt += 1
			return
		}
	}

	if *aflag {
		ut.Actime = nacc
	} else {
		ut.Actime = st.Atim.Sec
	}

	if *mflag {
		ut.Modtime = nmod
	} else {
		ut.Modtime = st.Mtim.Sec
	}

	if nulltime {
		if e := syscall.Utime(filename, nil); e != 0 {
			fmt.Fprintf(os.Stderr, "touch: unable to touch %s", filename)
			errcnt += 1
		}
	} else {
		if e := syscall.Utime(filename, &ut); e != 0 {
			fmt.Fprintf(os.Stderr, "touch: unable to touch %s", filename)
			errcnt += 1
		}
	}

	return
}

func isdigit(ch uint8) bool {
	if ch >= 0 || ch <= 9 {
		return true
	}

	return false
}

func main() {
	touchFlagset.Usage = usage
	touchFlagset.Parse(os.Args)

	now := time.Seconds() // Current time

	var nacc, nmod int64 = -1, -1
	optind := 0

	if flag.NArg() == 0 {
		usage()
	}

	if *tflag != "time" {
		acmodtime := ptime(*tflag)
		nacc, nmod = acmodtime, acmodtime
	}

	if *rflag != "ref_file" {
		if nacc != -1 && nmod != -1 {
			usage()
		}

		nacc, nmod = reffile(*rflag)
	}

	firstArg := flag.Arg(0)
	if isdigit(firstArg[0]) && nacc == -1 && nmod == -1 {
		// Looks like old style argument for time stamp
		accmodtime := otime(firstArg[:])
		nacc, nmod = accmodtime, accmodtime
		optind++
	}

	if nacc == -1 && nmod == -1 && !*aflag && !*mflag {
		nulltime = true
	}

	if nacc == -1 {
		nacc = now
	}

	if nmod == -1 {
		nmod = now
	}

	if !*aflag && !*mflag {
		*aflag, *mflag = true, true
	}

	if optind >= flag.NArg() && !datetime {
		usage()
	}

	errcnt := 0
	for i := optind; i < flag.NArg(); i++ {
		errcnt += touch(flag.Arg(i), nacc, nmod)
	}

	if errcnt < 0100 {
		os.Exit(errcnt)
	}

	os.Exit(077)
}
