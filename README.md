# enos

Enos is the forthcoming QTI CLI application. For now, it mostly includes [reference
Terraform workspaces](/workspaces/enos-modules-dev) that utilize the `enos-provider`
and `terraform-enos-*` Terraform modules.

***

## General tips

### Save temporary Doormat credentials to local AWS credentials file

- If you don’t already have active Doormat credentials, run

    `doormat --refresh`
- You will need your AWS account name, account ID, and access level. You can do this by checking them at https://doormat.hashicorp.services/, or by running `doormat aws --list` to see your eligible roles on the accounts you have access to.

- Now, run the following, replacing `<account_number>`, `<account_name>`, and `<access_level>` with yours:
 
    `doormat aws --role arn:aws:iam::<account_number>:role/<account_name>-<access_level> -m`

    (`-m` = manage (for AWS configs))

- To check if it was successful:

    `cd ~/.aws`
    
    `cat credentials`

- It should show your updated `aws_access_key_id`, `aws_secret_access_key`, and `aws_session_token`.


