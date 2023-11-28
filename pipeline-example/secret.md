
私有仓库 secret

1 根据官方文档获取 个人的 token

2 创建 对应的secret：

```
kubectl create secret generic git-credentials \
--namespace=kurator-pipeline \
--from-literal=.gitconfig=$'[credential "https://github.com"]\n\thelper = store' \
--from-literal=.git-credentials='https://Xieql:xxx@github.com'
```

cosign secret 创建 secret，包含了 私钥 key 公钥pub
需要用户输入两次 password 用来作为私钥获取凭证。为了测试，可以直接输入两次空格

cosign generate-key-pair k8s://tekton-chains/signing-secrets 

