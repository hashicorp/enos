default: init run destroy

init: verify-license-init
run: verify-license-run
destroy: verify-license-destroy

verify-license-init:
	export TF_CLI_CONFIG_FILE=./mirror.tfrc &&\
	mkdir -p ./terraform-plugin-cache && \
	terraform -chdir=./scenarios/verify-license init -upgrade -var-file=../../terraform.tfvars &&\
	chmod -R +x ./terraform-plugin-cache/hashicorp.com/qti/enos/*

verify-license-run:
	export TF_CLI_CONFIG_FILE=./mirror.tfrc &&\
	terraform -chdir=./scenarios/verify-license apply -var-file=../../terraform.tfvars -auto-approve

verify-license-destroy:
	export TF_CLI_CONFIG_FILE=./mirror.tfrc &&\
	terraform -chdir=./scenarios/verify-license destroy -var-file=../../terraform.tfvars -auto-approve
