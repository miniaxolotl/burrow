# Changelog

## [0.3.5](https://github.com/miniaxolotl/burrow/compare/v0.3.4...v0.3.5) (2026-10-20)


### Features

* **website:** refresh Burrow Arc identity and Plex design kit ([#31](https://github.com/miniaxolotl/burrow/issues/31)) ([9fa9c00](https://github.com/miniaxolotl/burrow/commit/9fa9c001faab7731ac9b6474ddad60b7456f74f3))

## [0.3.4](https://github.com/miniaxolotl/burrow/compare/v0.3.3...v0.3.4) (2026-10-20)


### Bug Fixes

* **release:** align shared CI/CD workflows and reproducible packing ([2cb7768](https://github.com/miniaxolotl/burrow/commit/2cb7768acd1f78a86460d097e59f479f475220a9))
* **release:** preserve packed permissions across runner umasks ([e5dccfb](https://github.com/miniaxolotl/burrow/commit/e5dccfb4b995a5330708c291b0aea61659112113))

## [0.3.3](https://github.com/miniaxolotl/burrow/compare/v0.3.2...v0.3.3) (2026-10-20)


### Bug Fixes

* **release:** preserve executable and reproducible npm artifacts ([#25](https://github.com/miniaxolotl/burrow/issues/25)) ([53e9c74](https://github.com/miniaxolotl/burrow/commit/53e9c7403c15801930620d4bb9ba60006cc861d4))

## [0.3.2](https://github.com/miniaxolotl/burrow/compare/v0.3.1...v0.3.2) (2026-10-20)


### Bug Fixes

* **release:** wait for npm publication propagation ([#22](https://github.com/miniaxolotl/burrow/issues/22)) ([cc3771d](https://github.com/miniaxolotl/burrow/commit/cc3771d7bac56e1cd4274928106a86dcc8cf3668))

## [0.3.1](https://github.com/miniaxolotl/burrow/compare/v0.3.0...v0.3.1) (2026-10-20)


### Features

* **website:** add Vercel Speed Insights ([#18](https://github.com/miniaxolotl/burrow/issues/18)) ([b471880](https://github.com/miniaxolotl/burrow/commit/b47188019d530cf2a266e634bab58e8c3c5ed94b))
* **website:** align Burrow developer site with Bramble Labs branding ([2f65efb](https://github.com/miniaxolotl/burrow/commit/2f65efb16a78c5d6a28e69d23dc0b23e45c5e5d5))
* **website:** polish burrow identity, journeys and policies ([#17](https://github.com/miniaxolotl/burrow/issues/17)) ([70dbbc4](https://github.com/miniaxolotl/burrow/commit/70dbbc41b7cbac48dea5d5c2925e7012acd8eab8))
* **website:** refine Bramble family styling and product navigation ([b76185c](https://github.com/miniaxolotl/burrow/commit/b76185cb7d28b16663b6b5288848a8b96642efe6))


### Bug Fixes

* **release:** gate production publication on frozen source candidates ([#20](https://github.com/miniaxolotl/burrow/issues/20)) ([b851a4f](https://github.com/miniaxolotl/burrow/commit/b851a4f743986120a14177982a8edd29ee8ab3fc))

## [0.3.0](https://github.com/miniaxolotl/burrow/compare/v0.2.8...v0.3.0) (2026-10-16)


### ⚠ BREAKING CHANGES

* upgrade burrowctl and burrowd together for the consolidated owner-only API and commands. Legacy anonymous tunnel IDs are retired; auth commands and shared admin secrets are removed.

### Bug Fixes

* address ownership PR review findings ([f037847](https://github.com/miniaxolotl/burrow/commit/f037847030af080e6e29f2b7ea8a2c443234ca21))
* enforce private owner-only tunnel management ([dd73e59](https://github.com/miniaxolotl/burrow/commit/dd73e596b3ff1446a78810048871d0c738d57d35))

## [0.2.8](https://github.com/miniaxolotl/burrow/compare/v0.2.7...v0.2.8) (2026-10-16)


### Features

* **client:** replace default HTTP client with custom client and set read limit for WebSocket messages ([c4b2846](https://github.com/miniaxolotl/burrow/commit/c4b2846ab5b5acae5764a2c3adb927b4f6cb6212))
* **deploy:** manage Burrow environments with OpenTofu ([f59671f](https://github.com/miniaxolotl/burrow/commit/f59671f8b0084be93e488125e40f50155c921f5f))
* **deploy:** manage Burrow environments with OpenTofu ([0004e10](https://github.com/miniaxolotl/burrow/commit/0004e105fe893d026a27ff76f9232bc30c89344e))
* **docs:** add comprehensive Stripe best practices and integration guides ([76a075c](https://github.com/miniaxolotl/burrow/commit/76a075c799c9a4bb42396d8c9e7671cb509f3d4d))
* **docs:** add UI images to README for Tunnel Manager section ([3f90557](https://github.com/miniaxolotl/burrow/commit/3f90557dc97e2c6a0a175487e4b637bca7aeabd7))
* **server:** set read limit for WebSocket connections ([2ba6e81](https://github.com/miniaxolotl/burrow/commit/2ba6e8100952da54db0567d2121e0510ef0b2526))
* **skills:** add initial skills configuration for Stripe best practices and projects ([7c0aa6b](https://github.com/miniaxolotl/burrow/commit/7c0aa6b475fe506e2b6d410fea0c7f7a44d7c5da))
* **website:** add polished Bramble Labs marketing site ([b811081](https://github.com/miniaxolotl/burrow/commit/b8110814f2d34e0116d61904807e196fb46b4d80))
* **website:** add polished Bramble Labs marketing site ([49c83d3](https://github.com/miniaxolotl/burrow/commit/49c83d30a569fd744dfbb7f9cc28e5195c96a17f))
* **website:** enable Vercel Web Analytics ([a82943a](https://github.com/miniaxolotl/burrow/commit/a82943ace424b0c07ae3f1451f466c399023bb97))
* **website:** enable Vercel Web Analytics ([f8d1f2e](https://github.com/miniaxolotl/burrow/commit/f8d1f2e0bb0b09d0789a10b06b79967613c7f443))


### Bug Fixes

* **deploy:** honor Upstash budget minimum and verify reconnects ([1fa2176](https://github.com/miniaxolotl/burrow/commit/1fa217681523880a5c5a99939ee874997e07216f))
* **deploy:** honor Upstash budget minimum and verify reconnects ([6378d4a](https://github.com/miniaxolotl/burrow/commit/6378d4a4dd671a4a7d60f159e1e94e395097f7d7))
* **release:** isolate release preparation and check publishing access ([11c45d4](https://github.com/miniaxolotl/burrow/commit/11c45d4bd42170195714b2b6231f7f37804b95c3))
* **release:** isolate release preparation and check publishing access ([d9a4408](https://github.com/miniaxolotl/burrow/commit/d9a44083fbbdde5783705f400061280ba75d43c9))
* **server:** improve handling of tunnel cancellation and cleanup logic ([c27b8e4](https://github.com/miniaxolotl/burrow/commit/c27b8e487f72522b7b42daef39fe32991c45b42a))
* **server:** preserve HTTP request bytes after hijacking ([34fb0b5](https://github.com/miniaxolotl/burrow/commit/34fb0b5feffe286c75fec5cbf9adb0cd92a8c3a5))
* **server:** preserve HTTP request bytes after hijacking ([e17d1d9](https://github.com/miniaxolotl/burrow/commit/e17d1d980720ce4300240c3ee444e61e62a08b88))

## [0.2.7](https://github.com/miniaxolotl/burrow/compare/v0.2.6...v0.2.7) (2026-04-26)


### Features

* add set-server command to configure burrow server address and update default server in login ([868639d](https://github.com/miniaxolotl/burrow/commit/868639d2075fe1db0a72c40cf8f8863f6b0399a7))
* enhance tunnel registration with owner authentication and update related methods ([815e7b0](https://github.com/miniaxolotl/burrow/commit/815e7b03ec206940ab843ded23fe516606f9d2a4))


### Bug Fixes

* allow tunnel creation without authentication by always returning true in authTunnel ([2fa8dc1](https://github.com/miniaxolotl/burrow/commit/2fa8dc10b09491fe6f4e09e13b442c0ce7d81225))
* update default server address to burrow.mawa.dev in client configuration ([a11936f](https://github.com/miniaxolotl/burrow/commit/a11936ffce75f129f6e72c62e8224dde2f9ef096))
* update deploy workflow to resolve tags and improve version handling ([5c93b8c](https://github.com/miniaxolotl/burrow/commit/5c93b8cffa6ca6d8d52e54d1dcfa5f96da296f34))
* update preview version handling in deploy workflow ([81f7589](https://github.com/miniaxolotl/burrow/commit/81f75896e7bffbf17fd9c01d08026e030520e639))

## [0.2.6](https://github.com/miniaxolotl/burrow/compare/v0.2.5...v0.2.6) (2026-04-26)


### Bug Fixes

* update optionalDependencies in package.json and pnpm-lock.yaml to use workspace references ([7e1472f](https://github.com/miniaxolotl/burrow/commit/7e1472fa59b8447d8e1433e0f2c34354453307a6))

## [0.2.5](https://github.com/miniaxolotl/burrow/compare/v0.2.4...v0.2.5) (2026-04-26)


### Bug Fixes

* refactor release script to use fs.readFileSync for package.json access and improve path handling ([2cf2afa](https://github.com/miniaxolotl/burrow/commit/2cf2afa461cf1620790e8b74373e7ba111b4a498))
* update deployment and release workflows for improved version handling and Docker integration ([fdd81b0](https://github.com/miniaxolotl/burrow/commit/fdd81b048497f29c7634911322bd2291996c7836))
* update ESLint and TypeScript config descriptions to reflect burrow monorepo ([5529aed](https://github.com/miniaxolotl/burrow/commit/5529aedd3dede5df13c7400a04d72ffe24e24893))
* update release workflow to trigger on development branch ([100c013](https://github.com/miniaxolotl/burrow/commit/100c01353e134a845c21c4e0e145083fc2a1c8b8))

## [0.2.4](https://github.com/miniaxolotl/burrow/compare/v0.2.3...v0.2.4) (2026-04-26)


### Bug Fixes

* add optionalDependencies for burrowctl platform binaries ([5e8029c](https://github.com/miniaxolotl/burrow/commit/5e8029c7a860e41bce47f584592b7a546d199fdf))

## [0.2.3](https://github.com/miniaxolotl/burrow/compare/v0.2.2...v0.2.3) (2026-04-26)


### Features

* add build and deployment scripts, configure npm publishing, and implement GoReleaser for multi-platform binaries ([6031497](https://github.com/miniaxolotl/burrow/commit/6031497eb0e4d4f6796035e2a39471f2c038be8e))
* add build arguments and labels for versioning in Docker build process ([67d9f7f](https://github.com/miniaxolotl/burrow/commit/67d9f7f82f0d940056521ef30939061de7efa6fc))
* add build script for compiling burrowctl and burrowd binaries ([2f8f551](https://github.com/miniaxolotl/burrow/commit/2f8f551f08ac792330e9ec1bcd96d98d8357d0af))
* add client IP tracking to request logs and update documentation ([2282b83](https://github.com/miniaxolotl/burrow/commit/2282b83f61e2fe210ad22a7f10a3fac7e7fbae52))
* add deployment and installation scripts, update build process for burrowctl ([c227fdc](https://github.com/miniaxolotl/burrow/commit/c227fdc31c09fd92a6a5353a099724b22a04089c))
* add Docker Hub badge to README ([91837a2](https://github.com/miniaxolotl/burrow/commit/91837a2d17cebcb04a3b035ff6328fa78b28330e))
* add fast navigation and paging shortcuts to TUI and update documentation ([3dcab12](https://github.com/miniaxolotl/burrow/commit/3dcab1226c325629dce37e86e9cc22020f3ae7e1))
* add GitHub Actions workflow for deployment to Dokku ([a6a82da](https://github.com/miniaxolotl/burrow/commit/a6a82dab530b42456293bc88edc551e10eb7752d))
* add logging functionality and enhance tunnel management ([b7b3dff](https://github.com/miniaxolotl/burrow/commit/b7b3dff0f91ed651807ed2656ddc52440aaed352))
* add metadata labels to Dockerfile for improved documentation and source tracking ([6a05d7c](https://github.com/miniaxolotl/burrow/commit/6a05d7ce14e682ce09949b8fc2d6a20de0e8b3e4))
* add pull-requests permission to release job in workflow ([a2bc002](https://github.com/miniaxolotl/burrow/commit/a2bc002c4b7cad74e91a9bdd1fe1739b9af22e8d))
* add reconnection and graceful shutdown ([1afeb98](https://github.com/miniaxolotl/burrow/commit/1afeb985dfc2c3efaefcc887e76215f473199a67))
* add release-please workflow and update release.yml for improved release management ([b820442](https://github.com/miniaxolotl/burrow/commit/b820442c47c8276818dd46006834132813888d70))
* add shared ESLint and TypeScript configurations for the monorepo; update deployment and release scripts ([0aceb6a](https://github.com/miniaxolotl/burrow/commit/0aceb6ab3a8a292db4c4a7ba80eb3c52c2133b3c))
* add ssh_host_key to deploy steps for Dokku in CI workflow ([210c5a1](https://github.com/miniaxolotl/burrow/commit/210c5a16e9e281ff7704f1fd7c8e8d6015b8823c))
* add total size tracking for tunnels in server and update TunnelInfo struct ([bc37a1a](https://github.com/miniaxolotl/burrow/commit/bc37a1aeda65f3e8c1509e90dba00b4cd7c3ee44))
* add unlock step for dokku preview deployment ([f81a8b1](https://github.com/miniaxolotl/burrow/commit/f81a8b1bfdd3a6dccf08dbb5998edc2964dd2689))
* add version management to burrow and burrowd with SetVersion function ([c1a148e](https://github.com/miniaxolotl/burrow/commit/c1a148e7797d1f0e5a5ed4275d7383689221bbbb))
* adjust table width calculations in TUI for improved layout and rendering ([c621e91](https://github.com/miniaxolotl/burrow/commit/c621e91c2e58bb2471f421758e7744599dc0a829))
* consolidate CI/CD workflows by adding CI configuration and removing separate deploy workflow ([3c1d95b](https://github.com/miniaxolotl/burrow/commit/3c1d95b889279a4a9c305ec01c100bfcc0cccc28))
* enhance CI/CD workflows with specific paths for Go commands, add Docker Hub credential check, and improve version handling in release process ([d01f77b](https://github.com/miniaxolotl/burrow/commit/d01f77b53836789fd2496f380c978abc69b36403))
* enhance log view with IP address display and improve help section structure ([15fa770](https://github.com/miniaxolotl/burrow/commit/15fa7707b1d9b608ac838521efa0bf01cb4f9d24))
* enhance log viewing with auto-follow status and improved help display ([19ce582](https://github.com/miniaxolotl/burrow/commit/19ce582228d4940754b0cf0967b9139a99b45203))
* enhance TUI with sorting, confirmation prompts, and help overlay ([7109df2](https://github.com/miniaxolotl/burrow/commit/7109df20035a90e5b3379d458c45bc4403e99651))
* enhance tunnel ID extraction documentation to support multiple domain depths ([6cef9f9](https://github.com/miniaxolotl/burrow/commit/6cef9f974281b49f33a5b49d5399c7b58ac2356c))
* enhance tunnel management with context cancellation and reconnection logic ([a08535f](https://github.com/miniaxolotl/burrow/commit/a08535fe5ffdec56bdcb5789f5da129ff696e99a))
* expand idAdverbs and idAdjectives for more diverse tunnel ID generation ([c5ded4c](https://github.com/miniaxolotl/burrow/commit/c5ded4c37d22fed178b45bd286a8622fc20aeb0b))
* implement authentication commands with login, logout, and status functionalities ([e19e378](https://github.com/miniaxolotl/burrow/commit/e19e3789b5281be073e6559d7f3cc19123e14d9a))
* implement CI/CD workflows for build, test, and deployment processes ([629fae1](https://github.com/miniaxolotl/burrow/commit/629fae1f380c6f44f9e2ae26220dba4be2087a14))
* implement countWriter for tracking total data size in tunnels and enhance client IP retrieval logic ([2796230](https://github.com/miniaxolotl/burrow/commit/2796230e31a30b262b0deac2fa77a16b75705097))
* implement TLS support for secure tunnel connections and update configuration ([a1d4471](https://github.com/miniaxolotl/burrow/commit/a1d447178526231a2a5788ea38f6dad65ad23757))
* implement tunnel grace period for reconnecting clients and manage pending removals ([80c31ee](https://github.com/miniaxolotl/burrow/commit/80c31eee7c96d18b6ff94f54395769bb538efd7b))
* implement tunnel inspect and list functionalities with logging support ([a9b4162](https://github.com/miniaxolotl/burrow/commit/a9b416235c01f8c7239218b7fc734d66d42620a5))
* improve tunnel creation error handling and add cleanup for deleted tunnels ([314eb16](https://github.com/miniaxolotl/burrow/commit/314eb16540e2e6e8bbc3141de84b9f0695174a02))
* refactor TUI banner rendering, improve error handling in server responses, and optimize tunnel data retrieval ([523da70](https://github.com/miniaxolotl/burrow/commit/523da70cf87274f9be98d997926bea09e89e8187))
* replace xtaci/yamux with hashicorp/yamux and update related code for improved functionality ([1752219](https://github.com/miniaxolotl/burrow/commit/1752219341fae8c634f88f11e43fdc6523ea3fc2))
* restructure package directories and add platform-specific binaries for burrowctl ([bdbea54](https://github.com/miniaxolotl/burrow/commit/bdbea5440883f4b126bd5623d40a985a3ccf1675))
* streamline CI workflow by removing build job and updating deployment steps ([d177e74](https://github.com/miniaxolotl/burrow/commit/d177e74f2640169dfcc897da116a1610fa293a39))
* update .gitignore to include all .env files and enhance server documentation with additional domain examples ([4deb52d](https://github.com/miniaxolotl/burrow/commit/4deb52dedd1e18daee087288e1fd59f94046b167))
* update CI workflow for consistent deployment conditions and add deploy script ([b0a91f4](https://github.com/miniaxolotl/burrow/commit/b0a91f4a263f520e748b6e6b6a0286d29016f064))
* update CI workflows to specify directories for go vet, build, and test commands; add go work sync step in release process ([0386a65](https://github.com/miniaxolotl/burrow/commit/0386a65fe86a1253a870194ead34369c55dea336))
* update client and server documentation to include TLS support and new configuration options ([3b20030](https://github.com/miniaxolotl/burrow/commit/3b200301e92b312d69cae9ee8877e22048f1562f))
* update domain references from inkspire.one to mawa.dev across configuration and documentation ([fdfa06f](https://github.com/miniaxolotl/burrow/commit/fdfa06f2c8d2e054c467ecd75c457d0968aeb8ce))
* update package.json files for burrowctl binaries and add optional dependencies ([11a5b52](https://github.com/miniaxolotl/burrow/commit/11a5b521ee7e8bbd37be982fc9901e20bccd24e3))
* update ParseTunnelID function to include domain validation for tunnel ID extraction ([f07bbfd](https://github.com/miniaxolotl/burrow/commit/f07bbfd62ebfed7554093636b4e3b25bdba18517))
* update release workflow to build with goreleaser and improve platform-specific binary handling ([3a4fe8b](https://github.com/miniaxolotl/burrow/commit/3a4fe8bda55a47607eace349c5c8f88a8d8307a4))


### Bug Fixes

* add setup-go and goreleaser install to release workflow ([94ddacf](https://github.com/miniaxolotl/burrow/commit/94ddacf7cfb73114ea0919dd93ab4f96006cbf54))
* always install goreleaser, remove optionalDependencies update ([f2ff799](https://github.com/miniaxolotl/burrow/commit/f2ff79921a71e6c497af1ac8b0a94ce5c0cab2d6))
* correct arm64 binary name, ldflags package path ([6e18424](https://github.com/miniaxolotl/burrow/commit/6e184243f41c6bbf72f0cba0d70e5e9521dfef08))
* correct arm64 binary paths in release script (v8.0 suffix) ([be21d9e](https://github.com/miniaxolotl/burrow/commit/be21d9e476aed11c4d3179222ee638dac6d8ff8a))
* correct git push command syntax and improve goreleaser installation check in release script ([3be8253](https://github.com/miniaxolotl/burrow/commit/3be82536be47cd99f8ad12f80072d68ab5f1e487))
* correct remote URL secrets in CI workflow for deployment ([7f928d4](https://github.com/miniaxolotl/burrow/commit/7f928d481a383f193d9df34c860a7bd84c2ee1cb))
* deploy script tags images with version (e.g. v0.2.0), version number (0.2.0), and latest ([cdb7cf3](https://github.com/miniaxolotl/burrow/commit/cdb7cf3669fd6b3f3903712d236b3e3c32b2b46a))
* dokku deployment ([969febb](https://github.com/miniaxolotl/burrow/commit/969febbbb54688d139f1c220de87a58fa531d7bf))
* enhance help rendering by adjusting spacing in TUI output ([e91e009](https://github.com/miniaxolotl/burrow/commit/e91e00943ac4b43e51c7719b6c2989e09768767f))
* ensure branch is specified for both preview and production deployments in CI workflow ([da9cac8](https://github.com/miniaxolotl/burrow/commit/da9cac86eadc1dd48fd24d65e8f1e6ab458e0979))
* ensure shift command handles no arguments in deploy script ([12dde52](https://github.com/miniaxolotl/burrow/commit/12dde526a5210cfb491a4054bb17a5299c714c32))
* improve response messages in handleRoot and handleWebSocket functions ([1fed659](https://github.com/miniaxolotl/burrow/commit/1fed65989decec89967ee661606743668e803220))
* mask all but last 4 chars of token in auth status ([a221a83](https://github.com/miniaxolotl/burrow/commit/a221a83b120a403ca095d0af7a8a2d178be04492))
* nginx HTTP→HTTPS redirect, HSTS, wildcard domain ([e3d832d](https://github.com/miniaxolotl/burrow/commit/e3d832d6549b08266bf11c8deb38c87861a08720))
* remove broken unlock step from ci.yml (dokku action handles it) ([b3340e6](https://github.com/miniaxolotl/burrow/commit/b3340e6fe4b93781cd2ac7c8d77bc35f09f7eae0))
* remove optional dependencies for burrowctl packages in pnpm-lock.yaml ([f415a12](https://github.com/miniaxolotl/burrow/commit/f415a1255730a02ba9c5c41b51ae57f01bef304e))
* resolve critical race conditions and reliability bugs ([8a781bd](https://github.com/miniaxolotl/burrow/commit/8a781bd20c8ff2b64972142aae0d119c0ec3a52c))
* resolve goroutine deadlock in proxy and strip port from Host header ([c621e91](https://github.com/miniaxolotl/burrow/commit/c621e91c2e58bb2471f421758e7744599dc0a829))
* rewrite release/deploy workflows, fix scripts, update README ([9586132](https://github.com/miniaxolotl/burrow/commit/9586132cc924ceb146778c1c04bf31c2cbc29e2e))
* simplify tunnel creation by removing logger dependency and updating README for clarity ([1454608](https://github.com/miniaxolotl/burrow/commit/14546083c5da452c3445bae237b32c7f8ad15025))
* standardize quotes in version retrieval command in workflows ([1223a47](https://github.com/miniaxolotl/burrow/commit/1223a478d2d9f7e75a5ea08d5ff3a1a20e725536))
* streamline goreleaser installation and build command in release script ([d1e7b64](https://github.com/miniaxolotl/burrow/commit/d1e7b644a6b1157fbb7b84df7e61b61e98b122c1))
* update burrowctl package versions to 0.2.0 in pnpm-lock.yaml ([e17aa91](https://github.com/miniaxolotl/burrow/commit/e17aa9181013d0f03f30cf1f6bc3ff486174586b))
* update fallback domain in README and adjust token storage details in client/server documentation ([4111fad](https://github.com/miniaxolotl/burrow/commit/4111fad899a662679936192bf385984ed5ced4e7))
* update goreleaser installation command to remove version suffix ([5aeb586](https://github.com/miniaxolotl/burrow/commit/5aeb5866623f265ac047233f4fc4fe580f093f48))
* update goreleaser installation command to use correct path format ([92cf642](https://github.com/miniaxolotl/burrow/commit/92cf6423e4ce1fed65829bf0a67dc0512f4bf3ac))
* update package versions to 0.2.0 and adjust release script for correct binary path ([b04ccd5](https://github.com/miniaxolotl/burrow/commit/b04ccd516822ef4faff2fc0ddbc77f30b0c511fa))
* update tunnel domain from inkspire.app to inkspire.one in documentation ([b514a54](https://github.com/miniaxolotl/burrow/commit/b514a543947d533b410f3ebc385ee801c904f396))
* update tunnel IDs in client and server documentation for consistency ([796edf1](https://github.com/miniaxolotl/burrow/commit/796edf1444f7f491c59089e53f3efc5f88e8b054))
* update tunnel IDs in client documentation for accuracy ([f9df076](https://github.com/miniaxolotl/burrow/commit/f9df0767d883850887fafaf2a4be3b3803539121))
* update usage instructions and improve tunnel creation documentation ([40ac323](https://github.com/miniaxolotl/burrow/commit/40ac323c49dc45f8c174890c8b2973203b726127))
