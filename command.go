package main

type CommandMode interface {
	String() string
	Entered(string, App)
	Prompt() string
}
