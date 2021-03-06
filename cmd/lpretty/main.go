package main

import (
	"bufio"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/vrischmann/logfmt"
	"github.com/vrischmann/logfmt/internal"
)

const transformOperator = "::"

func extractTransform(args []string) (transform, []string) {
	if flMerge {
		return newMergeToJSONTransform(args), nil
	}

	//

	var res transforms

	if len(args) == 0 {
		return &stripKeyTransform{}, args
	}

	for _, arg := range args {
		t := newSinglePairTransform(arg)
		switch {
		case t != nil:
			res = append(res, t)
		default:
			res = append(res, &stripKeyTransform{key: arg})
		}

		args = args[1:]
	}

	return res, args
}
func runMain(cmd *cobra.Command, args []string) error {
	transform, args := extractTransform(args)

	//

	inputs := internal.GetInputs(args)

	buf := make([]byte, 0, 4096)
	for _, input := range inputs {
		scanner := bufio.NewScanner(input.Reader)
		for scanner.Scan() {
			line := scanner.Text()
			pairs := logfmt.Split(line)

			//

			result := transform.Apply(pairs)
			if result == nil {
				continue
			}

			switch v := result.(type) {
			case logfmt.Pairs:
				if len(v) == 0 {
					break
				}

				for _, pair := range v {
					buf = append(buf, []byte(pair.Value)...)
				}
				buf = append(buf, '\n')

			case []byte:
				buf = append(buf, v...)
				buf = append(buf, '\n')

			default:
				panic(fmt.Errorf("invalid result type %T", result))
			}

			//

			_, err := os.Stdout.Write(buf)
			if err != nil {
				return err
			}

			buf = buf[:0]
		}
		if err := scanner.Err(); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	rootCmd.Execute()
}

var (
	rootCmd = &cobra.Command{
		Use:   "lpretty [field[::[transform]]",
		Short: "prettify logfmt fields",
		Long: `prettify logfmt fields by applying transformation on them.

Every transform needs to be provided like this:
  <field name>::<transform name>

You can also just provide the field name and in this case the key from the field will be stripped from the output.

Transformations available:
  - json        for fields which contain valid JSON data
  - exception   for fields which contain valid Java exceptions

Examples:

    $ echo 'request="{\"id\":10,\"name\":\"Vincent\"}"' > /tmp/logfmt
    $ cat /tmp/logfmt | lpretty request::json
    {
        "request": {
            "id": 10,
            "name": "Vincent"
        }
    }

There is a second mode where the fields are merged using the flag --merge/-M. In this mode every field listed in the arguments
will be added to a single JSON object.

You can also indicate that a field contains valid JSON data in this mode so that the JSON is embedded directly in the object and
not reencoded as a string value.

Examples:

    $ echo 'id=10 name=vincent surname=Rischmann age=55' > /tmp/logfmt
    $ cat /tmp/logfmt | lpretty -M id age
    {
        "id": "10",
        "age": "55"
    }

    $ echo 'id=10 name=vincent data="{\"limit\":5000,\"offset\":30,\"table\":\"almanac\"}"' > /tmp/logfmt
    $ cat /tmp/logfmt | lpretty -M id data::json
    {
        "id": "10",
        "data: {
            "limit": 5000,
            "offset": 30,
            "table": "almanac"
        }
    }`,
		Args: cobra.MinimumNArgs(1),
		RunE: runMain,
	}

	flMerge bool
)

func init() {
	rootCmd.Flags().BoolVarP(&flMerge, "merge", "M", false, "Merge all fields in a single JSON object")
}
