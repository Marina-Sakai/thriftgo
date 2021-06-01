// Copyright 2021 CloudWeGo
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/thriftgo/generator"
	"github.com/cloudwego/thriftgo/generator/backend"
	"github.com/cloudwego/thriftgo/plugin"
)

// StringSlice implements the flag.Value interface on string slices
// to allow a flag to be set multiple times.
type StringSlice []string

func (ss *StringSlice) String() string {
	return fmt.Sprintf("%v", *ss)
}

// Set implements the flag.Value interface.
func (ss *StringSlice) Set(value string) error {
	*ss = append(*ss, value)
	return nil
}

// Arguments contains command line arguments for thriftgo.
type Arguments struct {
	AskVersion bool
	Recursive  bool
	Verbose    bool
	Quiet      bool
	OutputPath string
	Includes   StringSlice
	Plugins    StringSlice
	Langs      StringSlice
	IDL        string
}

// Output returns an output path for generated codes for the target language.
func (a *Arguments) Output(lang string) string {
	if len(a.OutputPath) > 0 {
		return a.OutputPath
	}
	return "./gen-" + lang
}

// UsedPlugins returns a list of plugin.Desc for plugins.
func (a *Arguments) UsedPlugins() (descs []*plugin.Desc, err error) {
	for _, str := range a.Plugins {
		desc, err := plugin.ParseCompactArguments(str)
		if err != nil {
			return nil, err
		}
		descs = append(descs, desc)
	}
	return
}

// Targets returns a list of generator.LangSpec for target languages.
func (a *Arguments) Targets() (specs []*generator.LangSpec, err error) {
	for _, lang := range a.Langs {
		desc, err := plugin.ParseCompactArguments(lang)
		if err != nil {
			return nil, err
		}

		spec := &generator.LangSpec{
			Language: desc.Name,
			Options:  desc.Options,
		}
		specs = append(specs, spec)
	}
	return
}

// MakeLogFunc creates logging functions according to command line flags.
func (a *Arguments) MakeLogFunc() backend.LogFunc {
	var logs = backend.LogFunc{}

	if a.Verbose && !a.Quiet {
		logger := log.New(os.Stderr, "[INFO] ", 0)
		logs.Info = func(v ...interface{}) {
			logger.Println(v...)
		}
	} else {
		logs.Info = func(v ...interface{}) {}
	}

	if !a.Quiet {
		logger := log.New(os.Stderr, "[WARN] ", 0)
		logs.Warn = func(v ...interface{}) {
			logger.Println(v...)
		}
		logs.MultiWarn = func(ws []string) {
			for _, w := range ws {
				logger.Println(w)
			}
		}
	} else {
		logs.Warn = func(v ...interface{}) {}
		logs.MultiWarn = func(ws []string) {}
	}

	return logs
}

// BuildFlags initializes command line flags.
func (a *Arguments) BuildFlags() *flag.FlagSet {
	f := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	f.BoolVar(&a.AskVersion, "version", false, "")

	f.BoolVar(&a.Recursive, "r", false, "")
	f.BoolVar(&a.Recursive, "recurse", false, "")

	f.BoolVar(&a.Verbose, "v", false, "")
	f.BoolVar(&a.Verbose, "verbose", false, "")

	f.BoolVar(&a.Quiet, "q", false, "")
	f.BoolVar(&a.Quiet, "quiet", false, "")

	f.StringVar(&a.OutputPath, "o", "", "")
	f.StringVar(&a.OutputPath, "out", "", "")

	f.Var(&a.Includes, "i", "")
	f.Var(&a.Includes, "include", "")

	f.Var(&a.Langs, "g", "")
	f.Var(&a.Langs, "gen", "")

	f.Var(&a.Plugins, "p", "")
	f.Var(&a.Plugins, "plugin", "")

	f.Usage = help
	return f
}

// Parse parse command line arguments.
func (a *Arguments) Parse(argv []string) {
	f := a.BuildFlags()
	if err := f.Parse(argv[1:]); err != nil {
		println(err)
		os.Exit(2)
	}

	if a.AskVersion {
		println("thriftgo", Version)
		os.Exit(0)
	}

	rest := f.Args()
	if len(rest) != 1 {
		println("require exactly 1 argument for the IDL parameter, got:", len(rest))
		os.Exit(2)
	}
	a.IDL = rest[0]
}

func help() {
	println("Version:", Version)
	println(`Usage: thriftgo [options] file
Options:
  --version           Print the compiler version and exit.
  -h, --help          Print help message and exit.
  -i, --include dir   Add a search path for includes.
  -o, --out dir	      Set the ouput location for generated files. (default: ./gen-*)
  -r, --recurse       Generate codes for includes recursively.
  -v, --verbose       Output detail logs.
  -q, --quiet         Suppress all warnings and informatic logs.
  -g, --gen STR       Specify the target langauge.
                      STR has the form language[:key1=val1[,key2[,key3=val3]]].
                      Keys and values are options passed to the backend.
                      Many options will not require values. Boolean options accept
                      "false", "true" and "" (emtpy is treated as "true").
  -p, --plugin STR    Specify an external plugin to invoke.
                      STR has the form plugin[=path][:key1=val1[,key2[,key3=val3]]].

Available generators (and options):
`)
	// print backend options
	for _, b := range g.AllBackend() {
		name, lang := b.Name(), b.Lang()
		println(fmt.Sprintf("  %s (%s):", name, lang))
		println(align(b.Options()))
	}
	println()
	os.Exit(2)
}

// align the help strings for plugin options.
func align(opts []plugin.Option) string {
	var names, descs, ss []string
	var max = 0
	for _, opt := range opts {
		names = append(names, opt.Name)
		descs = append(descs, opt.Desc)
		if max <= len(opt.Name) {
			max = len(opt.Name)
		}
	}

	for i := range names {
		rest := 2 + max - len(names[i])
		ss = append(ss, fmt.Sprintf(
			"    %s:%s%s",
			names[i],
			strings.Repeat(" ", rest),
			descs[i],
		))
	}
	return strings.Join(ss, "\n")
}