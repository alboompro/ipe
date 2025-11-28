// Copyright 2014 Claudemiro Alves Feitosa Neto. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package main provides the main entry point for the IPE application.
package main

import (
	"flag"
	"fmt"

	"ipe"
)

// Main function, initialize the system
func main() {
	var filename = flag.String("config", "config.yml", "Config file location")
	flag.Parse()

	printBanner()

	ipe.Start(*filename)
}

// Print a beautiful banner
func printBanner() {
	// Gradient colors from Alboom brand - pink to purple to teal to cyan
	// Using true color (24-bit) ANSI escape codes
	r := "\033[0m"

	// Brand colors
	pink := "\033[38;2;238;53;142m"  // #EE358E
	purple := "\033[38;2;95;52;148m" // #5F3494
	teal := "\033[38;2;85;191;171m"  // #55BFAB
	cyan := "\033[38;2;94;194;238m"  // #5EC2EE

	fmt.Println()

	// Alboom Ipê - using box drawing for clean look
	fmt.Printf(" %s╔═╗%s %s╦%s   %s╔╗%s  %s╔═╗%s %s╔═╗%s %s╔╦╗%s   %s╦%s %s╔═╗%s %s╔═╗%s\n",
		pink, r, pink, r, purple, r, purple, r, teal, r, teal, r, cyan, r, cyan, r, cyan, r)
	fmt.Printf(" %s╠═╣%s %s║%s   %s╠╩╗%s %s║ ║%s %s║ ║%s %s║║║%s   %s║%s %s╠═╝%s %s╠╣%s\n",
		pink, r, pink, r, purple, r, purple, r, teal, r, teal, r, cyan, r, cyan, r, cyan, r)
	fmt.Printf(" %s╩ ╩%s %s╩═╝%s %s╚═╝%s %s╚═╝%s %s╚═╝%s %s╩ ╩%s   %s╩%s %s╩%s   %s╚═╝%s\n",
		pink, r, pink, r, purple, r, purple, r, teal, r, teal, r, cyan, r, cyan, r, cyan, r)

	fmt.Println()
}
