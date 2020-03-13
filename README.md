[![CircleCI](https://circleci.com/gh/kouzoh/merlin.svg?style=svg&circle-token=5c9140edab4f649c6f3585fde235e63e093dd791)](https://circleci.com/gh/kouzoh/merlin)

# Merlin

An agent that sends out alerts when kubernetes resources are misconfigured or have issues based on rules.

## Technical background and designs

You can find the technical backgrounds and design in [ERD](https://docs.google.com/document/d/1KB0cSwG6b_h9vW5qpq-am9YFDYod3sRLBHpY4gF4m68/edit#)

## Install

### Install CRDs
In order to install merlin, you'll need to install the CRD (custom resource definition) first, run

```bash
make install
```  
to install the CRDs



### Uninstall CRDs
To uninstall CRDs, you can run:
```bash
make uninstall
```


### Sample custom resources
Once CRDs are installed, you can start setting what rules you'd like to have, there are several samples under `config/samples`, 
you can run 
```bash
make apply-samples
```
to apply all sample resources


### Install controller manager
After you've applied the custom resources, run
```bash
make deploy
``` 
to install the controller.

Note this will create a namespace called `merlin-us-dev` and install the controller manager in it.

## Testing
You can run the tests with 
```bash
make test
```

By default this uses [`envtest`](https://github.com/kubernetes-sigs/controller-runtime/tree/master/pkg/envtest)
to run the tests, but you can change it to use existing cluster by setting environment varialbe `USE_EXISTING_CLUSTER=1`

for references, [here](https://github.com/kubernetes-sigs/controller-runtime/blob/528cd19ee0de5d4732234566f756ef75f8c5ce77/pkg/envtest/server.go#L37-L45) 
is the list of environment variables can be used to change `envtest`'s behavior.

## Debugging with existing cluster
You can also run the controllers locally against an existing cluster that your kube config points to, just run 
```bash
make run
``` 



