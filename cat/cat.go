// cat - catenate files
//
// Code is pretty much line for line with the golang.org website's example 
// rotting cats.  Not a while lot can be changed to get a more simple 
// implementation.
package main

import (
	"os"
	"flag"
	"fmt"
)

var (
	sflag = flag.Bool("s",false,"Be silent about errors. File open/read etc")
)

var silentOnErr = false

func cat(f *os.File) {
	const NBUF = 8192
	var buf [NBUF]byte

	for {
		switch nr, rerr := f.Read(buf[:]);{
		case nr > 0:
			var nw, werr = os.Stdout.Write(buf[0:nr])
			if nw != nr && ! silentOnErr{
				fmt.Fprintf(
					os.Stderr,
					"cat: write error copying %s: %s",
					f.Name(),
					werr.String())
				os.Exit(1)
			}
		case nr == 0:
			return
		case nr < 0:
			fmt.Fprintf(
				os.Stderr,
				"cat: error reading %s: %s",
				f.Name(),
				rerr.String())
			os.Exit(1)
		}
	}
}

func main() {
	flag.Parse()

	if *sflag {
		silentOnErr = true
	}
	
	if flag.NArg() > 0 {
		for i := 0; i < flag.NArg(); i++ {
			var f, err = os.Open(flag.Arg(i))
			if f == nil && ! silentOnErr {
				fmt.Fprintf(
					os.Stderr,
					"cat: can't open %s: %s",
					flag.Arg(i),
					err.String())
				os.Exit(1)
			}
			cat(f)
			f.Close()
		}
	} else {
		cat(os.Stdin)
	}
}
