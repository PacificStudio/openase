# Changelog

## [0.8.0](https://github.com/PacificStudio/openase/compare/v0.7.0...v0.8.0) (2026-04-30)


### Features

* add provider delete flow ([57cc325](https://github.com/PacificStudio/openase/commit/57cc325b0991f0a85d75828ce7b389b4806c02d5))
* **codex:** add gpt-5.5 as default model ([baaa632](https://github.com/PacificStudio/openase/commit/baaa6328a292915c65fe940d04b889c5bb8fb971))
* **provider:** add provider delete flow ([c6c4e34](https://github.com/PacificStudio/openase/commit/c6c4e34843fa6fd16b6720f9046869c1db6c203a))


### Bug Fixes

* **activity:** normalize catalog labels ([9ca9dbd](https://github.com/PacificStudio/openase/commit/9ca9dbdb6d64cb5e80d6b5a062f6fccc60cf663a))
* **chat:** drain remote terminal output before exit ([76e3f9d](https://github.com/PacificStudio/openase/commit/76e3f9d417e4b85ff12e2abf4f20dadcd30e328e))
* **chat:** mark initial workspace clone as preparing ([#755](https://github.com/PacificStudio/openase/issues/755)) ([fca69ca](https://github.com/PacificStudio/openase/commit/fca69cab7e98bea69e24a158179f35243926bfb5))
* clean up zh copy and parse settings display text ([#753](https://github.com/PacificStudio/openase/issues/753)) ([30f822b](https://github.com/PacificStudio/openase/commit/30f822b3b301369ee631fce5d46da49c3a353f5a))
* **cli:** expose provider delete command ([add2364](https://github.com/PacificStudio/openase/commit/add236408b83783dcc07093b0f265d059f6afb9b))
* **deps:** upgrade pgx to v5.9.0 ([2d92cb3](https://github.com/PacificStudio/openase/commit/2d92cb32ffdba0e1ebd563c4dd852d8fb4c4312d))
* **httpapi:** sanitize legacy project ai scopes on patch ([1391419](https://github.com/PacificStudio/openase/commit/139141907a3c7ac4cfd36b2006926c80c416d9c1))
* **machine:** dedupe local registrations and probes ([8fab813](https://github.com/PacificStudio/openase/commit/8fab813a561d28dadb80e1fc2aa3461c9e5885ff))
* **machines:** satisfy lint after maintenance refactor ([6a550e8](https://github.com/PacificStudio/openase/commit/6a550e84c88ab157a0d18d3e12a7992aebe52e62))
* **orchestrator:** narrow pending completion summary query ([c0c5961](https://github.com/PacificStudio/openase/commit/c0c59611adc87ba023ce015569a01c0cffbc76ac))
* **orchestrator:** sanitize runtime persistence text ([fd12c80](https://github.com/PacificStudio/openase/commit/fd12c802b117893fe5c53fb1c42a55aab20d98a5))
* sanitize runtime command output before persistence ([1a02041](https://github.com/PacificStudio/openase/commit/1a02041ca4bea8f626b4a805ef4fe709ac4e213f))
* **settings:** restore provider delete type import ([cc8c80a](https://github.com/PacificStudio/openase/commit/cc8c80a9aa522833bcd3ccdf2e591d6f537cc732))
* upgrade pgx to v5.9.0 for Dependabot alert [#38](https://github.com/PacificStudio/openase/issues/38) ([df3e651](https://github.com/PacificStudio/openase/commit/df3e6512f4a5c26a65b004e990eae102ffd01630))
* **web:** localize provider delete dialog ([1c88c1e](https://github.com/PacificStudio/openase/commit/1c88c1e07be79c9b3e73638f983188723b64094c))
* **workspace:** upgrade go-git redirect protection ([7abba82](https://github.com/PacificStudio/openase/commit/7abba82e54bf799f1f8efa9d46bb4ee578247ae8))

## [0.7.0](https://github.com/PacificStudio/openase/compare/v0.6.0...v0.7.0) (2026-04-16)


### Features

* **chat:** add project ai permission guidance ([382a4d8](https://github.com/PacificStudio/openase/commit/382a4d84ea694a6a94f744e45069955db5773270))
* **chat:** align remote workspace capabilities ([e09784c](https://github.com/PacificStudio/openase/commit/e09784c2e863404136fe872435b9cec4929a3209))
* **chat:** refocus project ai prompt on planning ([dd9c7dc](https://github.com/PacificStudio/openase/commit/dd9c7dcc5119721461dc79d6b77e6209a6abe125))
* **machines:** add SSH bootstrap and improve machine UX ([#739](https://github.com/PacificStudio/openase/issues/739)) ([e8aa2b3](https://github.com/PacificStudio/openase/commit/e8aa2b3d8477a3ab89d7b33af9514830f06251eb))
* **platform:** add scoped project updates access ([6ec2455](https://github.com/PacificStudio/openase/commit/6ec245547c1fa972dc39272016bfc7113564914e))
* route HR adviser workflow suggestions through Project AI ([#744](https://github.com/PacificStudio/openase/issues/744)) ([cfedaf4](https://github.com/PacificStudio/openase/commit/cfedaf401746975c7d863eb5b88e39c5adc84a69))


### Bug Fixes

* **chat:** format restore helper updates ([585c2e2](https://github.com/PacificStudio/openase/commit/585c2e2bfc8210dcd27a1013cfd396ec7a04e002))
* **chat:** preserve terminal ready event ordering ([afae599](https://github.com/PacificStudio/openase/commit/afae599d1e26c0cf1f63b758fcbeb54282c7424c))
* **chat:** satisfy project conversation owner lint ([efc8a8e](https://github.com/PacificStudio/openase/commit/efc8a8e9cd2f5ed0166bcb47241d840f2502e515))
* **chat:** stabilize project conversation owners ([249621e](https://github.com/PacificStudio/openase/commit/249621ecce997a7ecf57dd16ef04f4ed1d95011d))
* **desktop:** patch follow-redirects resolution ([0210692](https://github.com/PacificStudio/openase/commit/0210692f16c9923f810ecdcdce324bd79b48c257))
* **machine-runtime:** scope agent cli paths by adaptor ([d53f5e2](https://github.com/PacificStudio/openase/commit/d53f5e2e7bb65e4855d313c57b579e2e5ad027ae))
* **onboarding:** localize GitHub connect copy ([3f5c74c](https://github.com/PacificStudio/openase/commit/3f5c74ca79c4fc28538957f550547dcffd78a1a0))
* scope machine agent CLI paths by adaptor ([f5faeb2](https://github.com/PacificStudio/openase/commit/f5faeb2134d5f7650d9a6a942b4c70c87ec61d4b))
* stabilize Project AI conversation owners ([f222f6e](https://github.com/PacificStudio/openase/commit/f222f6e6f7adbc1d574551b78ffb169d169de711))
* **ticket-runs:** scope list queries to the current ticket ([#737](https://github.com/PacificStudio/openase/issues/737)) ([34682c4](https://github.com/PacificStudio/openase/commit/34682c4b6abe32f362bbec74e90ddfe4dc75da33))
* **web:** format dompurify regression test ([320fb77](https://github.com/PacificStudio/openase/commit/320fb77ac3cadbbfff18cf6b6eef5adee708ce62))
* **web:** pin dompurify security upgrade ([8424605](https://github.com/PacificStudio/openase/commit/8424605ffb627f0aeed5708bf5a19842a77a3887))
* **web:** pin dompurify security upgrade ([1444e0a](https://github.com/PacificStudio/openase/commit/1444e0a1027515979716415c21aebeeb44be93d0))
* **web:** recover project views after reconnect ([f702903](https://github.com/PacificStudio/openase/commit/f7029036245b89a95931350c67026536daadb335))
* **web:** recover stale project views after reconnect ([13bd77b](https://github.com/PacificStudio/openase/commit/13bd77b6f807a6bed43e5c71751ef7e2999f8c9e))
* **web:** satisfy remote capability lint ([db4b5ae](https://github.com/PacificStudio/openase/commit/db4b5ae94afc20026be2d6c01d6fa5a98e91ecf4))

## [0.6.0](https://github.com/PacificStudio/openase/compare/v0.5.0...v0.6.0) (2026-04-15)


### Features

* **builtin:** add workflow planning authoring skills ([d2c9acd](https://github.com/PacificStudio/openase/commit/d2c9acd3abdcb81087e3372542b1df14abdeecd1))
* **cli:** support inline ticket repo scopes ([#730](https://github.com/PacificStudio/openase/issues/730)) ([f5abb0a](https://github.com/PacificStudio/openase/commit/f5abb0ab75571a2e57ad89724eb7d1e8d25257e8))
* **docs:** 添加核心概念和 Project AI 文档 ([c84af8b](https://github.com/PacificStudio/openase/commit/c84af8b931895c2a93ebd25417d785c083d73557))
* **doctor:** add HTTP port conflict detection ([7065e03](https://github.com/PacificStudio/openase/commit/7065e036955d2bf4563f215d3e54ae7d3519d129))
* **machine:** add layered websocket health telemetry ([#728](https://github.com/PacificStudio/openase/issues/728)) ([a3e8fb5](https://github.com/PacificStudio/openase/commit/a3e8fb568b17d072ec8324af67bf4919d72a2612))
* **machine:** support ssh topology bootstrap modes ([5a857ad](https://github.com/PacificStudio/openase/commit/5a857ad11e40acdeaba21a6911cb69fa0504be05))
* **orchestrator:** inject ticket context into workflow prompts ([af4d440](https://github.com/PacificStudio/openase/commit/af4d4408ba01d2b77f591f09e123472397bcfede))


### Bug Fixes

* catch up runtime pages after SSE reconnect ([b62e7e7](https://github.com/PacificStudio/openase/commit/b62e7e78d91509e1a04e3fe40d580a8587233dff))
* clarify Claude stop/resume execution errors ([#722](https://github.com/PacificStudio/openase/issues/722)) ([574ed66](https://github.com/PacificStudio/openase/commit/574ed66e5ad5d235694df65b22e9630be0999bed))
* **httpapi:** align project ticket reset routes ([5fb43c7](https://github.com/PacificStudio/openase/commit/5fb43c7448a405931eac202a6a874a52d3cd36d6))
* **httpapi:** expose project-scoped workspace reset ([caa1411](https://github.com/PacificStudio/openase/commit/caa1411ae1a63ee50f82c4a92dff1e5cefe8d0d1))
* patch desktop axios transitive dependency ([916111f](https://github.com/PacificStudio/openase/commit/916111f2ad78675c29714221f193181fc67af811))
* patch desktop wait-on axios dependency ([5086184](https://github.com/PacificStudio/openase/commit/5086184a217c03dcde651a7af7d7c34cb91e10b8))
* patch transitive cookie dependency ([b558ea6](https://github.com/PacificStudio/openase/commit/b558ea6854afb199dcaf8e08db2ede03b6a0e7fa))
* remove vulnerable minimatch from web lint deps ([94800e9](https://github.com/PacificStudio/openase/commit/94800e91b9db73625b9c46e8b84d255a464d508e))
* **web:** annotate picomatch 4.0.3 security guard ([143bf9f](https://github.com/PacificStudio/openase/commit/143bf9fd6594650ac9b2176a9f408e8afddf73a1))
* **web:** patch @sveltejs/kit security advisory ([9bbfa36](https://github.com/PacificStudio/openase/commit/9bbfa36a96912f7ec3cee4c6bac7e8b81e2d5515))
* **web:** pin lodash-es for streamdown markdown ([#698](https://github.com/PacificStudio/openase/issues/698)) ([a5bd7eb](https://github.com/PacificStudio/openase/commit/a5bd7eb8ca7891a9ec8c3cef5a05b41a034c5c4d))
* **web:** remove vulnerable picomatch lock resolution ([847c4fe](https://github.com/PacificStudio/openase/commit/847c4feec3e56943437a29c78d40b65fedce30e0))

## [0.5.0](https://github.com/PacificStudio/openase/compare/v0.4.0...v0.5.0) (2026-04-13)


### Features

* add workspace branch switching and git graph ([#679](https://github.com/PacificStudio/openase/issues/679)) ([e352a10](https://github.com/PacificStudio/openase/commit/e352a10c49e5218e1457b69497ed7d7ea2aa5dfd))
* **chat:** add multi-tab Project AI workspace file editor ([#666](https://github.com/PacificStudio/openase/issues/666)) ([3446f2f](https://github.com/PacificStudio/openase/commit/3446f2fbbcba2cea8827973934a348f0e0c9bb1c))
* **chat:** ship workspace browser v2 with path search ([f66b5ba](https://github.com/PacificStudio/openase/commit/f66b5bab3ac1791ad754b8b516f2c1c5322ee012))
* **skills:** add builtin auto-harness bundle ([#669](https://github.com/PacificStudio/openase/issues/669)) ([fbfe224](https://github.com/PacificStudio/openase/commit/fbfe22471e3a780e47a005d5d1161d51d832bb48))
* unify runtime raw/activity/transcript events ([c4e85a9](https://github.com/PacificStudio/openase/commit/c4e85a988d84e67095d8108aafb01b83b857f2a1))


### Bug Fixes

* **chat:** sanitize project conversation platform scopes ([71379b1](https://github.com/PacificStudio/openase/commit/71379b12f588d80c645920c9d3d60980622de676))
* **desktop:** bypass local browser auth for desktop runtime ([af76296](https://github.com/PacificStudio/openase/commit/af762968f6b8463f9d307933b807eb3e6f3c9209))
* restore Project AI fallback tabs and background queue flush ([ab9e4dd](https://github.com/PacificStudio/openase/commit/ab9e4dd3fa050da08ecf875d81888e2fc8e6fe6b))
* show ticket drawer run error details ([378baa5](https://github.com/PacificStudio/openase/commit/378baa53821a360255e1de32162d13ea9752979a))
* **ticket-detail:** normalize transcript history cursors ([e850fa7](https://github.com/PacificStudio/openase/commit/e850fa7cfe9f91287a21a3367db7472de4fd27a9))

## [0.4.0](https://github.com/PacificStudio/openase/compare/v0.3.0...v0.4.0) (2026-04-11)


### Features

* add a read-only Project AI workspace browser ([16dfbd9](https://github.com/PacificStudio/openase/commit/16dfbd99846d9f8bd42e39b7b88b0b97cef3f86a))
* add turn-level stop for Project AI conversations ([#651](https://github.com/PacificStudio/openase/issues/651)) ([f4d312e](https://github.com/PacificStudio/openase/commit/f4d312eb0bfbd0c676b0fa61f0418c674fef232b))
* **chat:** add local shell terminal foundation ([78bc163](https://github.com/PacificStudio/openase/commit/78bc163e1f8dbf0ec4f51cbff2cb33e17edeaeea))
* **chat:** add read-only workspace browser ([0a3b954](https://github.com/PacificStudio/openase/commit/0a3b954868494bf853b2fb1e3039875c0e63c72f))
* **chat:** support terminal detach and reattach ([f18b67d](https://github.com/PacificStudio/openase/commit/f18b67d94d7f2e71e814b50c406da5b0f02d26eb))
* enhance workspace browser with file comparison and navigation features ([70b4f29](https://github.com/PacificStudio/openase/commit/70b4f297722ea2c303b8e7cdfad3343bac1cb25a))
* **platform:** add bootstrap CLI auth and project AI scope controls ([c8bb830](https://github.com/PacificStudio/openase/commit/c8bb8305c44f1a6bab672670dab92294c63c349d))
* replace Textarea with CodeEditor in skill-file-editor for improved editing experience ([16ceef8](https://github.com/PacificStudio/openase/commit/16ceef8a07faf8686189d5dcf07c17605744549d))
* support Project AI conversation deletion and retention ([#659](https://github.com/PacificStudio/openase/issues/659)) ([584ba16](https://github.com/PacificStudio/openase/commit/584ba16748e87a721dd4eee76cba3d935703cebb))


### Bug Fixes

* auto-recover Project AI mux streams after transient disconnects ([de36057](https://github.com/PacificStudio/openase/commit/de360578b5a91ce75da1a43149e97fc16c76ab89))
* **chat:** prompt to sync newly bound repos before browse/diff ([dd0df8c](https://github.com/PacificStudio/openase/commit/dd0df8ceb930ea8e793a2919dc122683ab4b9bda))
* lock notification event coverage and wiring ([#657](https://github.com/PacificStudio/openase/issues/657)) ([6752cd3](https://github.com/PacificStudio/openase/commit/6752cd3a1837f894267cdfffa1350b04bf5b390f))
* make status/workflow CLI use platform-aware wrappers ([f3387c8](https://github.com/PacificStudio/openase/commit/f3387c8cd236ebc44b266c174d940f478d73543d))
* optimize scope draft creation in ticket-repo-scope-card for better state management ([16ceef8](https://github.com/PacificStudio/openase/commit/16ceef8a07faf8686189d5dcf07c17605744549d))
* serialize workspace init per machine ([#650](https://github.com/PacificStudio/openase/issues/650)) ([feb3734](https://github.com/PacificStudio/openase/commit/feb3734b4e2a9b4d48969a51c8a8d667b4866745))
* **web:** remove obsolete streamdown plugin shim ([5e3e998](https://github.com/PacificStudio/openase/commit/5e3e9985da91e584ebb1dd6d648582a9814c6c97))
* **web:** unblock workspace browser validation ([fb8e0cc](https://github.com/PacificStudio/openase/commit/fb8e0cc22cb9bbdf9e67715a7c3565423591528f))

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
