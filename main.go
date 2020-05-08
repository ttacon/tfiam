package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/zclconf/go-cty/cty"
)

var (
	f = flag.String("f", "", "file to parse")
)

func main() {
	flag.Parse()

	// Get the sources
	sources := getSources()
	fmt.Println(sources.debug())

	// Get AWS permissions for resources.
	_ = "foo"

	actions := awsActionsFromPermissions(sources)

	fmt.Println(actions)

	awsSession := session.New()
	stsSvc := sts.New(awsSession)
	caller, err := stsSvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Try the simulator, if it doesn't work, check our IAM permissions.
	svc := iam.New(awsSession)
	resp, err := svc.SimulatePrincipalPolicy(&iam.SimulatePrincipalPolicyInput{
		ActionNames:     actions,
		PolicySourceArn: caller.Arn,
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(resp)

	// Check our IAM permissions
	//
	// TODO(ttacon): document which IAM permissions are required to run
	// this.
	availablePermissions, err := getAvailableAWSPermissions()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println(availablePermissions)
}

func getSources() tfsources {
	file, err := os.Open(*f)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	var targets []string
	if fInfo, err := file.Stat(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else if !fInfo.IsDir() {
		targets = []string{*f}
	} else if found, err := file.Readdirnames(-1); err != nil {
		fmt.Println(err)
		os.Exit(1)
	} else {
		for _, fName := range found {
			if strings.HasSuffix(fName, ".tf") {
				targets = append(
					targets,
					filepath.Join(*f, fName),
				)
			}
		}
	}

	parser := hclparse.NewParser()
	for _, target := range targets {
		_, diags := parser.ParseHCLFile(target)
		for _, diag := range diags {
			if diag.Severity == hcl.DiagInvalid {
				continue
			}

			fmt.Println(diag)
			if diag.Severity == hcl.DiagError {
				os.Exit(1)
			}
		}
	}

	sources := tfsources{
		resources: map[string][]*hcl.Block{},
		data:      map[string][]*hcl.Block{},
		modules:   map[string][]*hcl.Block{},

		ctx: &hcl.EvalContext{
			Variables: make(map[string]cty.Value),
		},
	}

	for _, file := range parser.Files() {
		sources.processFile(file)
	}

	return sources
}

type resourceInfo struct {
	typ  string
	name string
}

// lifted from terraform 0.12 source
var terraformSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "terraform",
		},
		{
			Type:       "provider",
			LabelNames: []string{"name"},
		},
		{
			Type:       "variable",
			LabelNames: []string{"name"},
		},
		{
			Type: "locals",
		},
		{
			Type:       "output",
			LabelNames: []string{"name"},
		},
		{
			Type:       "module",
			LabelNames: []string{"name"},
		},
		{
			Type:       "resource",
			LabelNames: []string{"type", "name"},
		},
		{
			Type:       "data",
			LabelNames: []string{"type", "name"},
		},
	},
}
