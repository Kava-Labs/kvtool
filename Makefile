install:
	go install -ldflags "-X github.com/kava-labs/kvtool/config/generate.ConfigTemplatesDir=$(CURDIR)/config/templates"