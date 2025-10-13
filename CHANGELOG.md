# Changelog

## [1.10.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.9.0...v1.10.0) (2025-10-13)


### Features

* **jwt:** refactor equipment token generation API, update field names and encoding ([b64337a](https://github.com/tgym-digital/mini-rproxy/commit/b64337a9a487a03149a572fe2bd50345d9877d6d))

## [1.9.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.8.1...v1.9.0) (2025-10-13)


### Features

* **examples:** add gym-kit-settings.sh for fetching kit settings via curl ([2eae9aa](https://github.com/tgym-digital/mini-rproxy/commit/2eae9aa997d40b0b90b5c59af544ba67a19dcc81))


### Bug Fixes

* **rproxy:** use strings.CutPrefix for URL path modification and improve logging of path rewrite ([4a1caa6](https://github.com/tgym-digital/mini-rproxy/commit/4a1caa668893cfeea5efb53ce0c84e17da656cb1))

## [1.8.1](https://github.com/tgym-digital/mini-rproxy/compare/v1.8.0...v1.8.1) (2025-09-15)


### Bug Fixes

* forgot adding Bearer to jwt ([a7671da](https://github.com/tgym-digital/mini-rproxy/commit/a7671da429f226e86a59429f3357380555a2724e))

## [1.8.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.7.1...v1.8.0) (2025-09-10)


### Features

* **rproxy:** add support for upstream path mapping ([334cbd3](https://github.com/tgym-digital/mini-rproxy/commit/334cbd397798617263fc3ab1dd098a455944fea5))

## [1.7.1](https://github.com/tgym-digital/mini-rproxy/compare/v1.7.0...v1.7.1) (2025-09-10)


### Bug Fixes

* trying to fix this error? "You have an error in your yaml syntax on line 52" ([f1f546e](https://github.com/tgym-digital/mini-rproxy/commit/f1f546eb270c262a6978da12c9f0bbda6614c1f6))

## [1.7.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.6.1...v1.7.0) (2025-09-10)


### Features

* **docker:** successful docker image build with plugin load ([3b1df56](https://github.com/tgym-digital/mini-rproxy/commit/3b1df565da206332fb416329354733adef8d0c30))

## [1.6.1](https://github.com/tgym-digital/mini-rproxy/compare/v1.6.0...v1.6.1) (2025-09-10)


### Bug Fixes

* **plugin:** golang plugin not loading due to missing libc dep ([8c0a1f6](https://github.com/tgym-digital/mini-rproxy/commit/8c0a1f6e2d5c02a8b5b07f03115f46c47090efc2))

## [1.6.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.5.0...v1.6.0) (2025-09-10)


### Features

* **docker:** buildx based targeted builds ([093bce0](https://github.com/tgym-digital/mini-rproxy/commit/093bce07ff1402ac29070f1ec1bb8b70ff12057d))

## [1.5.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.4.0...v1.5.0) (2025-09-10)


### Features

* **dockerfile:** platform and cross compile opts ([6fa3d91](https://github.com/tgym-digital/mini-rproxy/commit/6fa3d91d879b34e209b00400a100e4e1227cdf5c))

## [1.4.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.3.0...v1.4.0) (2025-09-10)


### Features

* **docker:** cross-compile opts ([025844d](https://github.com/tgym-digital/mini-rproxy/commit/025844de7eef3531472778f0b3bb07e7fb86e42a))

## [1.3.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.2.0...v1.3.0) (2025-09-08)


### Features

* **plugins:** improve error handling and add docker commands ([ba02662](https://github.com/tgym-digital/mini-rproxy/commit/ba026620097fa2cb84af0676e8f37c67ed4fd3a5))

## [1.2.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.1.0...v1.2.0) (2025-09-05)


### Features

* **Dockerfile:** add plugindir param ([6abd7af](https://github.com/tgym-digital/mini-rproxy/commit/6abd7afcef59f9fcca9a02ffebfaec8ff407f85c))

## [1.1.0](https://github.com/tgym-digital/mini-rproxy/compare/v1.0.0...v1.1.0) (2025-09-05)


### Features

* **gha:** update release repo ([88ae146](https://github.com/tgym-digital/mini-rproxy/commit/88ae1464f4430567c2f5d34d3f09740d832b607b))
* **plugin:** simple plugins laoding system based of off https://pkg.go.dev/plugin ([aba5411](https://github.com/tgym-digital/mini-rproxy/commit/aba5411af412f23750405b2e7682b39016861532))

## 1.0.0 (2025-09-05)


### Features

* **gha:** initial release workflows ([3199e94](https://github.com/tgym-digital/rproxy/commit/3199e94484151ae88d52a0cd585bf7c7c37832b3))
* initial commit ([2d2b52a](https://github.com/tgym-digital/rproxy/commit/2d2b52a23267cd4ee2f1ada9962a18125e08f5f5))
* **mood shift:** this is now the tiniest reverse proxy know to man ([02dbcde](https://github.com/tgym-digital/rproxy/commit/02dbcdeb9473c4f8e15651f247b9a8e1cc91354e))
