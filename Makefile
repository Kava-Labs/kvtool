install:
	go install -ldflags "-X github.com/kava-labs/kvtool/cmd.ProjectDir=$(CURDIR)"