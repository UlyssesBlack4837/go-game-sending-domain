package main

import (
	"context"
	"flag"
	"fmt"
	"os"
)

const allowedTestRecipient = "chenhua@changba.com"

func main() {
	domain := flag.String("domain", "", "sending domain to register")
	to := flag.String("send-test-to", "", "optional recipient for a test message")
	flag.Parse()
	if *domain == "" {
		fmt.Fprintln(os.Stderr, "usage: go run . -domain mail.example.com [-send-test-to chenhua@changba.com]")
		os.Exit(2)
	}
	if *to != "" && *to != allowedTestRecipient {
		fatal(fmt.Errorf("-send-test-to must be %s", allowedTestRecipient))
	}

	client, err := NewClient()
	if err != nil {
		fatal(err)
	}
	ctx := context.Background()
	registration, err := client.VerifyDomain(ctx, *domain)
	if err != nil {
		fatal(err)
	}
	fmt.Println("publish these DNS records:")
	for _, record := range registration.DNSRecords {
		fmt.Printf("%s %s %s\n", record.Type, record.Name, record.Value)
	}

	status, err := client.Domain(ctx, *domain)
	if err != nil {
		fatal(err)
	}
	fmt.Println("verification status:", status.Verification.Status)

	if *to != "" {
		message, err := client.Send(ctx, *to, "Game backend domain check", "<p>Your game backend sending domain is configured.</p>")
		if err != nil {
			fatal(err)
		}
		fmt.Println("message id:", message.MessageID)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
