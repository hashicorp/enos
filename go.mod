module github.com/hashicorp/enos

go 1.22

// https://github.com/ryancragun/go-cty/commit/62c05dda59fe0f377ead06e0e2a59c2a643aa5df
replace github.com/zclconf/go-cty => github.com/ryancragun/go-cty v1.10.1-0.20240510213915-62c05dda59fe

require (
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/aws/aws-sdk-go-v2/config v1.18.39
	github.com/google/uuid v1.6.0
	github.com/hashicorp/go-hclog v1.6.3
	github.com/hashicorp/go-multierror v1.1.1
	github.com/hashicorp/hcl/v2 v2.21.0
	github.com/hashicorp/terraform-exec v0.21.0
	github.com/hashicorp/terraform-json v0.22.1
	github.com/hexops/gotextdiff v1.0.3
	github.com/mitchellh/cli v1.1.5
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db
	github.com/mitchellh/go-wordwrap v1.0.1
	github.com/olekukonko/tablewriter v0.0.5
	github.com/spf13/cobra v1.8.1
	github.com/stretchr/testify v1.8.4
	github.com/zclconf/go-cty v1.14.4
	github.com/zclconf/go-cty-yaml v1.0.3
	golang.org/x/net v0.27.0
	golang.org/x/sys v0.22.0
	golang.org/x/term v0.22.0
	golang.org/x/text v0.16.0
	google.golang.org/grpc v1.65.0
	google.golang.org/protobuf v1.34.2
)

require (
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/armon/go-radix v1.0.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.36.3 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.37 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.11 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.3.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.6.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.42 // indirect
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.224.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.12.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.12.15 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.13.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.15.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.21.5 // indirect
	github.com/aws/smithy-go v1.22.2 // indirect
	github.com/bgentry/speakeasy v0.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/google/go-cmp v0.6.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/posener/complete v1.2.3 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/crypto v0.25.0 // indirect
	golang.org/x/mod v0.19.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/tools v0.23.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240711142825-46eb208f015d // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
