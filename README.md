# Go La Tengo

Golang library for MySQL and MariaDB database automation

## Repo now archived

Due to some unfortunate circumstances, it has become necessary to permanently suspend development of this package as an independent repo.

Go La Tengo is integral to Skeema, but was designed as a reusable package in a separate repo, with the hope of building development momentum for use-cases *outside* of schema management. For example, this package's introspection logic could be used as the basis of a code-gen ORM. However, providing it as a separate repo requires a significant time investment: the codebase must be separately maintained, tested, versioned, and released; backwards-incompatible API changes must be avoided, limiting future refactors; and each change must be vendored inside of the Skeema CLI's repo as well.

In early 2021, one of the external users of this package created a hostile fork, for the sole purpose of creating schema management functionality which *directly* competes with Skeema. The forker is a startup that has raised over $100 million USD, but has nonetheless not contributed to this package (or any part of Skeema) in any way whatsoever. Worse still, their fork provides functionality which is only present in our Premium edition products.

Skeema is a bootstrapped (self-funded) product, and ongoing development is dependent on revenue from our Premium products. It is absolutely not practical or reasonable to continue development of this repo when our own code is effectively being used against us, by a competing company with much deeper pockets.

All functionality here has now been merged directly into the Skeema CLI's primary repo as an internal sub-package. As of 4 November 2021, this separate Tengo repo is now archived, and may be deleted entirely at some point in the future.

## External Dependencies

* https://github.com/go-sql-driver/mysql (Mozilla Public License 2.0)
* https://github.com/jmoiron/sqlx (MIT License)
* https://github.com/VividCortex/mysqlerr (MIT License)
* https://github.com/fsouza/go-dockerclient (BSD License)
* https://github.com/pmezard/go-difflib/difflib (BSD License)
* https://github.com/nozzle/throttler (Apache License 2.0)
* https://golang.org/x/sync/errgroup (BSD License)

## Credits

Created and maintained by [@evanelias](https://github.com/evanelias).

Additional [contributions](https://github.com/skeema/tengo/graphs/contributors) by:

* [@tomkrouper](https://github.com/tomkrouper)
* [@efixler](https://github.com/efixler)
* [@chrisjpalmer](https://github.com/chrisjpalmer)
* [@alexandre-vaniachine](https://github.com/alexandre-vaniachine)
* [@mhemmings](https://github.com/mhemmings)

Support for stored procedures and functions generously sponsored by [Psyonix](https://psyonix.com).

Support for partitioned tables generously sponsored by [Etsy](https://www.etsy.com).

## License

**Copyright 2021 Skeema LLC**

```text
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
```


