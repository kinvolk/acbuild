// Copyright 2015 The rkt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"crypto/sha512"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"text/template"

	"github.com/appc/acbuild/Godeps/_workspace/src/github.com/coreos/rkt/pkg/multicall"
	"github.com/appc/acbuild/Godeps/_workspace/src/github.com/spf13/cobra"

	"github.com/appc/acbuild/lib"
)

const (
	cliName = "acbuild"

	commandUsage = `\
NAME:
{{printf "\t%s - %s" .Name .Short}}

USAGE:
{{printf "\t%s" .UseLine}}

{{if (ne .Example "")}}\
EXAMPLE:
{{printf "\t%s" .Example}}

{{end}}\
\
{{if (ne (len .Commands) 0)}}\
COMMANDS:
{{range .Commands}}\
{{if (ne (len .Commands) 0)}}\
{{printf "\t%s%s\t%s" .Name (subcmdList .Commands) .Short}}
{{else}}\
{{printf "\t%s\t%s" .Name .Short}}
{{end}}\
{{end}}\

{{end}}\
\
OPTIONS:
{{.LocalFlags.FlagUsages}}`
)

var (
	debug       bool
	contextpath string
	aciToModify string

	cmdExitCode int

	templFuncs = template.FuncMap{
		"subcmdList": func(cmds []*cobra.Command) string {
			var subcmds []string
			for _, subcmd := range cmds {
				subcmds = append(subcmds, subcmd.Name())
			}
			return " [" + strings.Join(subcmds, "|") + "]"
		},
	}

	commandUsageTemplate = template.Must(template.New("command_usage").Funcs(templFuncs).Parse(strings.Replace(commandUsage, "\\\n", "", -1)))
)

var cmdAcbuild = &cobra.Command{
	Use:   "acbuild [command]",
	Short: "the application container build system",
}

func init() {
	cmdAcbuild.PersistentFlags().BoolVar(&debug, "debug", false, "Print out debug information to stderr")
	cmdAcbuild.PersistentFlags().StringVar(&contextpath, "work-path", ".", "Path to place working files in")
	cmdAcbuild.PersistentFlags().StringVar(&aciToModify, "modify", "", "Path to an ACI to modify (ignores build context)")

	cobra.EnablePrefixMatching = true
}

func newACBuild() *lib.ACBuild {
	return lib.NewACBuild(contextpath, debug)
}

func getErrorCode(err error) int {
	switch err {
	case lib.ErrNotFound:
		return 2
	case nil:
		return 0
	default:
		return 1
	}
}

// runWrapper return a func(cmd *cobra.Command, args []string) that internally
// will add command function return code and the reinsertion of the "--" flag
// terminator.
func runWrapper(cf func(cmd *cobra.Command, args []string) (exit int)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if aciToModify == "" {
			cmdExitCode = cf(cmd, args)
			return
		}

		command := strings.Split(cmd.Use, " ")[0]
		switch command {
		case "cat-manifest":
			cmdExitCode = runCatOnACI(aciToModify)
			return
		case "begin", "write", "end", "version":
			stderr("Can't use the --modify flag with %s.", command)
			cmdExitCode = 1
			return
		}

		finfo, err := os.Stat(aciToModify)
		switch {
		case os.IsNotExist(err):
			stderr("ACI doesn't appear to exist: %s.", aciToModify)
			cmdExitCode = 1
			return
		case err != nil:
			stderr("Error accessing ACI to modify: %v.", err)
			cmdExitCode = 1
			return
		case finfo.IsDir():
			stderr("ACI to modify is a directory: %s.", aciToModify)
			cmdExitCode = 1
			return
		}

		absoluteAciToModify, err := filepath.Abs(aciToModify)
		if err != nil {
			stderr("%v", err)
			cmdExitCode = 1
			return
		}

		hash := sha512.New().Sum([]byte(absoluteAciToModify))
		contextpath := path.Join(os.TempDir(), fmt.Sprintf("acbuild-%x", hash))

		if len(contextpath) > 16 {
			contextpath = contextpath[:16]
		}

		err = os.MkdirAll(contextpath, 0755)
		if err != nil {
			stderr("%v", err)
			cmdExitCode = 1
			return
		}
		defer os.RemoveAll(contextpath)

		a := newACBuild()

		err = a.Begin(aciToModify, false)
		if err != nil {
			stderr("%v", err)
			cmdExitCode = getErrorCode(err)
			return
		}

		cmdExitCode = cf(cmd, args)

		err = a.Write(aciToModify, true, false, nil)
		if err != nil {
			stderr("%v", err)
			cmdExitCode = getErrorCode(err)
			return
		}

		err = a.End()
		if err != nil {
			stderr("%v", err)
			cmdExitCode = getErrorCode(err)
			return
		}
	}
}

func main() {
	// check if acbuild is executed with a multicall command
	multicall.MaybeExec()

	cmdAcbuild.SetUsageFunc(func(cmd *cobra.Command) error {
		tabOut := new(tabwriter.Writer)
		tabOut.Init(os.Stdout, 0, 8, 1, '\t', 0)
		commandUsageTemplate.Execute(tabOut, cmd)
		tabOut.Flush()
		return nil
	})

	// Make help just show the usage
	cmdAcbuild.SetHelpTemplate(`{{.UsageString}}`)

	cmdAcbuild.Execute()
	os.Exit(cmdExitCode)
}

func stderr(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, strings.TrimSuffix(out, "\n"))
}

func stdout(format string, a ...interface{}) {
	out := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stdout, strings.TrimSuffix(out, "\n"))
}
