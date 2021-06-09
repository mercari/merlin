# Merlin

An agent that sends out alerts when kubernetes resources are misconfigured or have issues based on rules.

## Technical background and designs

Please see [Docs](https://github.com/mercari/merlin/tree/master/docs) for detailed documentations


## Installation

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

Note this will create a namespace called `merlin` and install the controller manager in it.


## Testing
You can run the tests with 
```bash
make test
```

By default controller tests use [`envtest`](https://github.com/kubernetes-sigs/controller-runtime/tree/master/pkg/envtest)
to run the tests, but you can change it to use existing cluster by setting environment varialbe `USE_EXISTING_CLUSTER=1`

for references, [here](https://github.com/kubernetes-sigs/controller-runtime/blob/528cd19ee0de5d4732234566f756ef75f8c5ce77/pkg/envtest/server.go#L37-L45) 
is the list of environment variables can be used to change `envtest`'s behavior.

Other tests uses [testing](https://golang.org/pkg/testing/) and [mock](https://github.com/golang/mock) for unit testing and mocking API calls. 

## Debugging with existing cluster
You can also run the controllers locally against an existing cluster that your kube config points to, just run 
```bash
make run
``` 


## Committers

 * Bill Chung ([@billcchung](https://github.com/billcchung))

## Contribution

Please read the CLA below carefully before submitting your contribution.

https://www.mercari.com/cla/

## License

Copyright 2021 Mercari, Inc.

Licensed under the Apache 2.0 License.
