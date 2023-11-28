
私有仓库 secret

根据官方文档获取 个人的 token


kubectl create secret generic git-credentials \
--namespace=kurator-pipeline \
--from-literal=.gitconfig=$'[credential "https://github.com"]\n\thelper = store' \
--from-literal=.git-credentials='https://Xieql:xxx@github.com'


cosign secret

