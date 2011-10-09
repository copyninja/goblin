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

var (
	aflag = flag.Bool("a", false, "Change the access time")
	mflag = flag.Bool("m", false, "Change the modification time")
	cflag = flag.Bool("c", false, "Don't create the file if not exists")
	tflag = flag.String("t", "time", "Use time instead of current time. Time specified as [[CC]YY]MMDDhhmm[.SS]")
	rflag = flag.String("r", "ref_file", "Use time of corresponding file rather than current time")
)

var (
	now      = time.Seconds() // Current time
	errcnt   = 0              // count errors
	nulltime = false          // no time given in the arguments
	datetime = true           // Use date_time operand not -t
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

func otime(cp string) syscall.Time_t {
	t := now
	stm := time.LocalTime()
	datetime = true

	stm.Second = 0

	switch len(cp) {
	case 10:
		if stm.Year = int64(atot(cp[8:],99)); stm.Year < 69 {
			stm.Year += 100
		}
		cp = cp[:8]
	case 8:
		stm.Minute = atot(cp[6:],59)
		cp = cp[:6]
		stm.Hour = atot(cp[4:],23)
		cp = cp[:4]
		if stm.Day = atot(cp[2:],31); stm.Day == 0 {
			badtime()
		}
		cp = cp[:2]

		if stm.Month = atot(cp,12); stm.Month == 0 {
			badtime()
		}
		stm.Month -= 1

	default:
		badtime()
	}

	t = stm.Seconds()
	return syscall.Time_t(t)

}

func ptime(cp string) syscall.Time_t {
	t := now
	stm := time.LocalTime()
	sz := len(cp)


	if sz == 11 || sz == 13 || sz == 15 {
		if strings.Index(cp, ".") != sz-3 {
			badtime()
		}

		stm.Second = atot(cp[sz-2:], 61)
		cp = cp[:sz-3]
		sz -= 3
	} else {
		stm.Second = 0
	}

	if sz == 12 {
		year := cp[:4]
		if stm.Year = int64(atot(year, 30000)); stm.Year < 1970 {
			badtime()
		}
		stm.Year -= 1900
		cp = cp[4:]
		sz -= 4
	} else if sz == 10 {
		year := cp[:2]
		if stm.Year = int64(atot(year, 99)); stm.Year < 69 {
			stm.Year += 100
		}

		cp = cp[2:]
		sz -= 2
	}

	if sz != 8 {
		badtime()
	}

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

	t = stm.Seconds()

	return syscall.Time_t(t)
}

func reffile(filename string) (syscall.Time_t, syscall.Time_t) {
	var st syscall.Stat_t

	if e := syscall.Stat(filename, &st); e != 0 {
		fmt.Fprintf(os.Stderr, "stat: Error %s\n", syscall.Errstr(e))
		os.Exit(1)
	}

	//FIXME: bug in Go! Amd64 has following rest have Atimespec and Mtimespec
	return syscall.Time_t(st.Atim.Sec), syscall.Time_t(st.Mtim.Sec)
}

func touch(filename string, nacc, nmod syscall.Time_t) {
	var st syscall.Stat_t
	var ut syscall.Utimbuf

	if e := syscall.Stat(filename, &st); e != 0 {
		if e == syscall.ENOENT {

			if *cflag {
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
		ut.Actime = int64(nacc)
	} else {
		ut.Actime = st.Atim.Sec
	}

	if *mflag {
		ut.Modtime = int64(nmod)
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

}

func main() {
	flag.Parse()

	nacc, nmod := syscall.Time_t(-1), syscall.Time_t(-1)
	optind := 0

	if flag.NArg() == 0 {
		usage()
	}

	if *tflag != "time" {
		acmodtime := ptime(*tflag)
		nacc, nmod = acmodtime, acmodtime
	}

	if *rflag != "ref_file" {
		if nacc != syscall.Time_t(-1) && nmod != syscall.Time_t(-1) {
			usage()
		}

		nacc, nmod = reffile(*rflag)
	}

	firstArg := flag.Arg(0)
	if firstArg[0] >= 0 || firstArg[0] <= 9 && nacc == syscall.Time_t(-1) && nmod == syscall.Time_t(-1) {
		// Looks like old style argument for time stamp
		accmodtime := otime(firstArg[:])
		nacc, nmod = accmodtime,accmodtime
		optind ++
	}

	if nacc == syscall.Time_t(-1) && nmod == syscall.Time_t(-1) && !*aflag && !*mflag {
		nulltime = true
	}

	if nacc == syscall.Time_t(-1) {
		nacc = syscall.Time_t(now)
	}

	if nmod == syscall.Time_t(-1) {
		nmod = syscall.Time_t(now)
	}

	if !*aflag && !*mflag {
		*aflag, *mflag = true, true
	}

	if optind >= flag.NArg() && ! datetime {
		usage()
	}
	
	for i := optind; i < flag.NArg(); i++ {
		touch(flag.Arg(i), nacc, nmod)
	}

	if errcnt < 0100 {
		os.Exit(errcnt)
	}

	os.Exit(077)
}
