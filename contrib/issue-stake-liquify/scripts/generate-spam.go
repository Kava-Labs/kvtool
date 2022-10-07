package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func usage() {
	fmt.Println("this script is used to generate a json file for input into issue-stake-liquify")
	fmt.Println("generate-spam <num-validators> <num-delegations> <min-amount> <max amount>")
	fmt.Println("  the script generates the `delegations` portion of the configuration")
	fmt.Println()
	fmt.Println("example: generate-spam 100 10000 100 1_000_000")
	fmt.Println("  this creates 10k delegations to 100 different validators between 100 & 1M KAVA")
}

func main() {
	if len(os.Args) != 5 {
		usage()
		fmt.Println("ERROR: unexpected number of arguments")
		os.Exit(1)
	}

	// parse args
	numValidators, err := strconv.ParseInt(os.Args[1], 10, 0)
	if err != nil {
		log.Fatalf("unable to parse numValidators: %s", err)
	}
	numDelegations, err := strconv.ParseInt(os.Args[2], 10, 0)
	if err != nil {
		log.Fatalf("unable to parse numDelegations: %s", err)
	}
	minAmount, ok := sdk.NewIntFromString(os.Args[3])
	if !ok {
		log.Fatal("unable to parse minAmount")
	}
	maxAmount, ok := sdk.NewIntFromString(os.Args[4])
	if !ok {
		log.Fatal("unable to parse maxAmount")
	}

	fmt.Println(numValidators)
	fmt.Println(numDelegations)
	fmt.Println(minAmount.String())
	fmt.Println(maxAmount.String())
}
