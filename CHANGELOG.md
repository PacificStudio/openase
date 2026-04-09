# Changelog

## [0.3.0](https://github.com/PacificStudio/openase/compare/v0.2.0...v0.3.0) (2026-04-09)


### Features

* add scoped secret management foundation ([30536fd](https://github.com/PacificStudio/openase/commit/30536fd2cf472930307112c5ecd9ca0a7c81b895))
* **board:** enhance board column with workflow pickup information and tooltip support ([091c67c](https://github.com/PacificStudio/openase/commit/091c67c370ed3b912ce266cebe677d2bfa463216))
* **dashboard:** add OrganizationProvidersSection and update onboarding logic ([3f7d0cb](https://github.com/PacificStudio/openase/commit/3f7d0cbaf72861cbf6c5dcbc8a19b8558033bb2c))
* **desktop:** add OpenASE desktop shell v1 ([#583](https://github.com/PacificStudio/openase/issues/583)) ([4381bfc](https://github.com/PacificStudio/openase/commit/4381bfca5377c26286d24b9bdb6f1214c2f950b1))
* generate multi-repo workspace instruction hubs ([#566](https://github.com/PacificStudio/openase/issues/566)) ([d439a66](https://github.com/PacificStudio/openase/commit/d439a669be17af22a7d8decf00bc7f8fb0e78c12))
* **iam:** add session governance and auth audit ([4554112](https://github.com/PacificStudio/openase/commit/4554112873acde0abe0308df00803d5ee3b10fec))
* **iam:** define dual-mode auth contract ([2082603](https://github.com/PacificStudio/openase/commit/2082603e9173f39ae13e226d6bdd0f7fed0116e5))
* **iam:** harden RBAC scope integrity ([deb2bba](https://github.com/PacificStudio/openase/commit/deb2bba512bf49fc7ded237ee8c8986fa5d7da9a))
* **iam:** ship the IAM admin console rollout flow ([db95c60](https://github.com/PacificStudio/openase/commit/db95c60525f32695a0ac35b10238c37b16c194fe))
* **mobile:** adapt Activity, Updates, Machines, Agents for phone/tablet ([#593](https://github.com/PacificStudio/openase/issues/593)) ([980d590](https://github.com/PacificStudio/openase/commit/980d59024de1c52173093c17258f97b5fb3b7323))
* **notifications:** redesign notification settings with grouped events and severity indicators ([1d00412](https://github.com/PacificStudio/openase/commit/1d0041273691343a27fe19e9c5c7b65e8082bcc7))
* **orchestrator:** generate multi-repo instruction hubs ([53e6f48](https://github.com/PacificStudio/openase/commit/53e6f48586de76cc6405b71366d54e0c157bb01b))
* **repository-editor:** enhance repository URL input with type switching and contextual help ([5be3a66](https://github.com/PacificStudio/openase/commit/5be3a661033b32ae1658c0bd70e056bfa67c60f6))
* **security:** add org GitHub credentials and board PR badges ([7f30ed9](https://github.com/PacificStudio/openase/commit/7f30ed9505922daee2c69b297c2f569f6c674622))
* **shell:** adapt project shell, navigation, search, and Project AI for mobile ([#590](https://github.com/PacificStudio/openase/issues/590)) ([a90b2fe](https://github.com/PacificStudio/openase/commit/a90b2fe56d66be5c50908d0daab0b7073efb1469))
* **web:** add ticket card context menu on board with archive action ([#570](https://github.com/PacificStudio/openase/issues/570)) ([071e06f](https://github.com/PacificStudio/openase/commit/071e06f348a5b029ea038ccb3acd3b2e4d92c1fa))
* **web:** refresh onboarding setup flow ([1326d03](https://github.com/PacificStudio/openase/commit/1326d03b7a30025a72836105d995dc8dffd71754))
* **web:** single-field composer and inline body display ([#568](https://github.com/PacificStudio/openase/issues/568)) ([51aaa49](https://github.com/PacificStudio/openase/commit/51aaa492b70d54deeb14feeb01d77825ece4ab32))


### Bug Fixes

* **ase-118:** align frontend contract usage ([436d7a6](https://github.com/PacificStudio/openase/commit/436d7a6f21ff4e505dc04135ea92bf52a34717b3))
* **ase-118:** restore access migration regression copy ([e0cf893](https://github.com/PacificStudio/openase/commit/e0cf893e9b5b9fba842870260fdeb0603e36c9f3))
* **chat:** remove queued project turn before dispatch ([2f912c5](https://github.com/PacificStudio/openase/commit/2f912c54a77f0cea20d5c7f99b25408341fe2d3d))
* **chat:** stabilize project conversation titles ([82fcad8](https://github.com/PacificStudio/openase/commit/82fcad8c331689e9b7c7ac6e2cd9414462a6f0c7))
* **ci:** quiet backend test logs in actions ([1757da5](https://github.com/PacificStudio/openase/commit/1757da532b33f75990d9685388402b4abc5925b9))
* **deploy:** preload agent CLIs in runtime image ([5eba202](https://github.com/PacificStudio/openase/commit/5eba2022249502c17ae38a3bc07b3c0e67cc76ea))
* **iam:** cover session CLI parity and trim settings UI ([18f7c63](https://github.com/PacificStudio/openase/commit/18f7c633997c53d4ee6968655321b919b973451c))
* **machinetransport:** stabilize preflight failure parsing ([767aa41](https://github.com/PacificStudio/openase/commit/767aa41cd097e61e9f620f07dfecffcc45ff7809))
* **orchestrator:** satisfy instruction hub lint gate ([cc74435](https://github.com/PacificStudio/openase/commit/cc744356ea6ccd428efc4705dea90d139aab1b6a))
* **orchestrator:** use root-scoped hub reads ([d7afd99](https://github.com/PacificStudio/openase/commit/d7afd992358f448358348eebdf1224bdbabb60d8))
* **repo:** support file project repo URLs ([d2985db](https://github.com/PacificStudio/openase/commit/d2985dba3806ceff53c336b85b195310051a9b37))
* shell-quote workflow hook runtime interpolation (ASE-59) ([e945239](https://github.com/PacificStudio/openase/commit/e94523973674dd95477a972b8fe8b052a78d623a))
* stabilize Project AI conversation ownership across origins ([#571](https://github.com/PacificStudio/openase/issues/571)) ([c7b4615](https://github.com/PacificStudio/openase/commit/c7b4615c79c2bb7596b994b2843ff152c2a73ef7))
* **ticket-detail:** replay queued runtime drawer refreshes ([778cdb5](https://github.com/PacificStudio/openase/commit/778cdb5e22b5c5753c69aa175d633a9552b35399))
* **web:** align session wrappers with frontend audit gate ([cbb4768](https://github.com/PacificStudio/openase/commit/cbb47687d36a5efb8adc85b8ccc46ea30b0f1fdc))
* **web:** remove nested shell scrolling with Project AI ([c605529](https://github.com/PacificStudio/openase/commit/c6055293a58bd2cf819273686ca5c08c8257c92b))
* **workspace:** handle unborn git heads in summaries ([1e27fb7](https://github.com/PacificStudio/openase/commit/1e27fb703aebc8cd69ce54c2efd6e6a742b3b4a7))

## [0.2.0](https://github.com/PacificStudio/openase/compare/v0.1.1...v0.2.0) (2026-04-05)


### Features

* **chat:** cross-project panel tabs with fixed ownership [ASE-32] ([6c1412b](https://github.com/PacificStudio/openase/commit/6c1412b91d1264fb091ff431d693257ceca4f203))
* **ui:** unify color picker around curated status palettes (ASE-54) ([#546](https://github.com/PacificStudio/openase/issues/546)) ([e2c6d5b](https://github.com/PacificStudio/openase/commit/e2c6d5b7a38cfd3fe44b65650cdcde1f70413370))
* unify websocket runtime execution contract ([a16ff19](https://github.com/PacificStudio/openase/commit/a16ff199a6816e1aaf98b08266ad26da8ba5cd18))


### Bug Fixes

* **chat:** preserve running tab state after hydration ([104f296](https://github.com/PacificStudio/openase/commit/104f29694bf88308d34dde35f4843cdf0dc59046))
* **ci:** avoid playwright port collisions ([1886e03](https://github.com/PacificStudio/openase/commit/1886e03fd9ab504de6d94a67e85b01f13ac88939))
* **ci:** avoid vite watch exhaustion in e2e ([c18f6af](https://github.com/PacificStudio/openase/commit/c18f6af81406d34a3d0811180f2cd7114bd09fa7))
* **ci:** repair prompt and streaming regressions ([885ebf0](https://github.com/PacificStudio/openase/commit/885ebf0c8138bccf71fbc230650078f37e7dddb6))
* **cli:** add generic body-contract parity coverage ([c6ad8c0](https://github.com/PacificStudio/openase/commit/c6ad8c048ada3c1a0d222fa2c0249b49f222908f))
* hot-refresh Project AI providers from org provider events ([1191f28](https://github.com/PacificStudio/openase/commit/1191f28f901e62687f2548370763249242326fde))
* **machines:** restore frontend CI for machine editor [ASE-43] ([15ee809](https://github.com/PacificStudio/openase/commit/15ee809f968d2b878a7d314756af17a558441076))
* **machines:** restore machine editor frontend CI ([fee327d](https://github.com/PacificStudio/openase/commit/fee327dab67ca7ebeb50ce18991aad8f8a97824c))
* **machines:** restore machine editor frontend CI ([15f6328](https://github.com/PacificStudio/openase/commit/15f6328939ebbf72613993739d012995545fd0aa))

## Changelog

All notable changes to this project will be documented in this file.

The release history is managed by Release Please from Conventional Commits.
