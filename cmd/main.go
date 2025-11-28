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
	// ANSI color codes for Alboom brand colors
	pink := "\033[38;5;205m"    // #EE358E
	purple := "\033[38;5;54m"   // #5F3494
	orange := "\033[38;5;215m"  // #FCBA72
	teal := "\033[38;5;72m"     // #55BFAB
	cyan := "\033[38;5;81m"     // #5EC2EE
	darkPurple := "\033[38;5;53m" // #462F80
	reset := "\033[0m"

	// Alboom logo in ASCII with brand colors
	fmt.Println()
	fmt.Printf("       %s██████%s▄▄%s\n", pink, purple, reset)
	fmt.Printf("       %s██████%s  ▀▀▄▄%s\n", pink, purple, reset)
	fmt.Printf("%s██████%s %s██████%s      ██%s\n", orange, reset, teal, purple, reset)
	fmt.Printf("%s██████%s %s██████%s      ██%s\n", orange, reset, teal, purple, reset)
	fmt.Printf("%s██████%s %s██████%s  ▄▄▀▀%s\n", orange, reset, teal, purple, reset)
	fmt.Printf("%s▄▄%s     %s██████%s▀▀%s\n", purple, reset, cyan, purple, reset)
	fmt.Printf("%s  ▀▀▄▄%s %s██████%s\n", purple, reset, cyan, reset)
	fmt.Printf("%s      ▀▀%s\n", purple, reset)
	fmt.Println()

	// Alboom Ipê text
	fmt.Printf("%s    _    _ _                           ___        __%s\n", darkPurple, reset)
	fmt.Printf("%s   /_\\  | | |__   ___   ___  _ __ ___  |_ _|_ __   ___  %s\n", darkPurple, reset)
	fmt.Printf("%s  //_\\\\ | | '_ \\ / _ \\ / _ \\| '_ ` _ \\  | || '_ \\ / _ \\ %s\n", darkPurple, reset)
	fmt.Printf("%s /  _  \\| | |_) | (_) | (_) | | | | | | | || |_) |  __/%s\n", darkPurple, reset)
	fmt.Printf("%s/_/ \\_\\_|_.__/ \\___/ \\___/|_| |_| |_||___|  __/ \\___|%s\n", darkPurple, reset)
	fmt.Printf("%s                                          |_|%s\n", darkPurple, reset)
	fmt.Println()
}
