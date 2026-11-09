# Claritty Security Policy

This policy covers the `clarctl` CLI, the Kubernetes SRE Agent and its deployment manifests, and the Hub API, dashboard, and published container image. Security fixes target the [latest stable release](https://github.com/Vaishnav88sk/claritty/releases/latest). Reports about older versions are welcome and should include the version tested.

## Report a vulnerability privately

Email the [Claritty maintainer](mailto:vaishnavsk8804@gmail.com?subject=Claritty%20security%20report) with the subject `Claritty security report`. Keep vulnerability details out of public issues, pull requests, and discussions while a fix is being coordinated.

Include as much of the following as you can.

- The affected component, version, image tag or digest, and relevant Kubernetes or deployment details.
- The permissions or configuration needed to reproduce the issue, along with the steps and observed impact.
- A minimal proof of concept or sanitized logs, if available. Replace API keys, kubeconfigs, database credentials, and sensitive cluster data with placeholders.

Test only in environments you own or have permission to test. The maintainer will review the report, ask for details if needed, and coordinate disclosure and any fix with you. Use the [issue tracker](https://github.com/Vaishnav88sk/claritty/issues) for ordinary bugs and the [contributing guide](CONTRIBUTING.md) for patches.
