# Understanding kustomize plugins
**Notes:** Much of the documentation is based on [kustomize.io](https://kubectl.docs.kubernetes.io/guides/extending_kustomize/)

## Extending kustomize
Kustomize offers a plugin framework allowing people to write their own resource generators and transformers.
Write a plugin when changing generator options or transformer configs doesn’t meet your needs.
+ A *generator* plugin emits all the components (deployment, service, ingress, etc.) to stdout.
+ A *transformer* plugin might perform special transformation beyond those provided by the builtin (namePrefix, commonLabels, etc.) transformers.

### Kinds of plugins
There are two kinds of plugins, Exec and Go shared library.

## Exec plugins
A exec plugin is any executable that accepts a single argument on its command line, the name of a YAML file containing its configuration. That file is provided by kustomize.<br>
*DataReplaceInline* is an exec plugin of transformer's family.
https://kubectl.docs.kubernetes.io/guides/extending_kustomize/execpluginguidedexample/

## Go plugins
A .go file can be a [Go plugin](https://golang.org/pkg/plugin/) if it declares ‘main’ as it’s package, and exports a symbol to which useful functions are attached.

https://kubectl.docs.kubernetes.io/guides/extending_kustomize/#go-plugins

## Generators family
A generator plugin accepts **nothing** on **stdin**, but emits generated resources to **stdout**.

## Transformers family
A transformer plugin accepts Kubernetes/OpenShift resources on **stdin** and emits those resources, presumably transformed, to **stdout**.

## Placement
Each plugin gets its own dedicated directory named
~~~
$XDG_CONFIG_HOME/kustomize/plugin/${apiVersion}/LOWERCASE(${kind})
~~~

The default value of XDG_CONFIG_HOME is $HOME/.config.

The one-plugin-per-directory requirement eases creation of a plugin bundle (source, tests, plugin data files, etc.) for sharing.

In the case of a Go plugin, it also allows one to provide a go.mod file for the single plugin, easing resolution of package version dependency skew.

When loading, kustomize will first look for an executable file called
~~~
$XDG_CONFIG_HOME/kustomize/plugin/${apiVersion}/LOWERCASE(${kind})/${kind}
~~~

If this file is not found or is not executable, kustomize will look for a file called ${kind}.so in the same directory and attempt to load it as a Go plugin.

If both checks fail, the plugin load fails the overall kustomize build.

## Execution
Plugins are only used during a run of the kustomize build command with the parameter --enable_alpha_plugins
~~~
kustomize build --enable_alpha_plugins kustomize_directory
~~~
