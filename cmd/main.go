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
	// Gradient colors from Alboom brand - hot pink to purple to cyan
	// Using true color (24-bit) ANSI escape codes for smooth gradients
	r := "\033[0m"

	// Color stops for gradient: Pink -> Purple -> Teal -> Cyan
	c := []string{
		"\033[38;2;238;53;142m", // #EE358E pink
		"\033[38;2;200;50;140m", // transition
		"\033[38;2;162;47;138m", // transition
		"\033[38;2;124;52;136m", // transition
		"\033[38;2;95;52;148m",  // #5F3494 purple
		"\033[38;2;85;140;160m", // transition
		"\033[38;2;85;175;165m", // transition
		"\033[38;2;85;194;171m", // #55BFAB teal
		"\033[38;2;94;194;238m", // #5EC2EE cyan
	}

	fmt.Println()
	fmt.Println()

	// ALBOOM IPÊ - Brutal gradient ASCII art
	fmt.Printf("  %s█████%s  %s██%s      %s████%s    %s█████%s   %s█████%s  %s███%s   %s███%s    %s████%s  %s████%s   %s█████%s\n",
		c[0], r, c[1], r, c[2], r, c[3], r, c[4], r, c[5], r, c[6], r, c[7], r, c[7], r, c[8], r)
	fmt.Printf(" %s██   ██%s %s██%s      %s██  ██%s  %s██   ██%s %s██   ██%s %s████%s %s████%s   %s██%s    %s██  ██%s %s██%s\n",
		c[0], r, c[1], r, c[2], r, c[3], r, c[4], r, c[5], r, c[6], r, c[7], r, c[7], r, c[8], r)
	fmt.Printf(" %s███████%s %s██%s      %s██████%s  %s██   ██%s %s██   ██%s %s██%s %s██%s %s██%s   %s██%s    %s██████%s %s████%s\n",
		c[0], r, c[1], r, c[2], r, c[3], r, c[4], r, c[5], r, c[5], r, c[6], r, c[7], r, c[7], r, c[8], r)
	fmt.Printf(" %s██   ██%s %s██%s      %s██   ██%s %s██   ██%s %s██   ██%s %s██%s     %s██%s   %s██%s    %s██%s      %s██%s\n",
		c[0], r, c[1], r, c[2], r, c[3], r, c[4], r, c[5], r, c[6], r, c[7], r, c[7], r, c[8], r)
	fmt.Printf(" %s██   ██%s %s██████%s %s████████%s %s█████%s   %s█████%s  %s██%s     %s██%s   %s████%s  %s██%s     %s█████%s\n",
		c[0], r, c[1], r, c[2], r, c[3], r, c[4], r, c[5], r, c[6], r, c[7], r, c[7], r, c[8], r)

	fmt.Println()
	fmt.Println()
}
