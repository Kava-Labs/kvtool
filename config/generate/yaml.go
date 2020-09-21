package generate

import (
	"io/ioutil"
	"os"

	"github.com/Jeffail/gabs/v2"
	"gopkg.in/yaml.v3"
)

func overwriteMergeYAML(sourceFileName, destinationFileName string) error {
	source, err := importYAML(sourceFileName)
	if err != nil {
		return err
	}
	destination, err := importYAML(destinationFileName)
	if err != nil {
		if os.IsNotExist(err) {
			destination = gabs.New()
		} else {
			return err
		}
	}
	err = destination.MergeFn(source, func(destination, source interface{}) interface{} {
		// overwrite any non-object values with the source's version
		return source
	})
	if err != nil {
		return err
	}
	if err := exportYAML(destinationFileName, destination); err != nil {
		return err
	}
	return nil
}

func importYAML(filename string) (*gabs.Container, error) {
	bz, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	unmarshalStructure := map[string]interface{}{}
	err = yaml.Unmarshal(bz, &unmarshalStructure)
	if err != nil {
		return nil, err
	}
	return gabs.Wrap(unmarshalStructure), nil
}

func exportYAML(filename string, data *gabs.Container) error {
	bz, err := yaml.Marshal(data.Data())
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filename, bz, 0644); err != nil {
		return err
	}
	return nil
}
