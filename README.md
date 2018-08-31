# lb-operator

# Usage
```
Create CRD
kubectl create -f example/albcrd.yaml
kubectl create -f example/clbcrd.yaml


Use CRD
kubectl get alb
kubectl get clb
```

# How to build
```
git clone git@github.com:sak0/lb-operator.git
cd lb-operator/cmd
CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o lb-operator
cd ..
docker build .
```
