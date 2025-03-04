package output

import (
	"io"
	"sort"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/salsadigitalauorg/shipshape/pkg/result"
)

type Outputter interface {
	Output(*result.ResultList) ([]byte, error)
}

var Outputters = map[string]Outputter{}

func RegistryKeys() []string {
	keys := []string{}
	for k := range Outputters {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func ParseConfig(raw map[string]interface{}, rl *result.ResultList) {
	count := 0
	log.WithField("registry", RegistryKeys()).Debug("outputters")
	for pluginName, pluginMap := range raw {
		o, ok := Outputters[pluginName]
		if !ok {
			continue
		}

		// Convert the map to yaml, then parse it into the plugin.
		// Not catching any errors here since the yaml content is known.
		pluginYaml, _ := yaml.Marshal(pluginMap)
		yaml.Unmarshal(pluginYaml, o)

		log.WithFields(log.Fields{"plugin": pluginName}).Debug("parsed outputter")
		count++
	}
	log.Infof("parsed %d outputters", count)
}

func OutputAll(rl *result.ResultList, w io.Writer) error {
	// Only write stdout outputter results to stdout
	if stdout, ok := Outputters["stdout"]; ok {
		buf, err := stdout.Output(rl)
		if err != nil {
			return err
		}

		if _, err := w.Write(buf); err != nil {
			return err
		}
	}

	// Process other outputters (like file) without writing to stdout
	for name, p := range Outputters {
		if name == "stdout" {
			continue
		}
		if _, err := p.Output(rl); err != nil {
			return err
		}
	}
	return nil
}
