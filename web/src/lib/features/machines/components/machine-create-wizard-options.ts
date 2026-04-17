import { ArrowLeftRight, ArrowRight, Monitor, Radio, Sparkles } from '@lucide/svelte'
import type {
  MachineWizardLocationAnswer,
  MachineWizardOptionCard,
  MachineWizardStrategy,
} from './machine-create-wizard-types'

export const machineWizardLocationOptions: MachineWizardOptionCard<MachineWizardLocationAnswer>[] =
  [
    {
      value: 'local',
      icon: Monitor,
      titleKey: 'machines.machineCreateWizard.location.local.title',
      descKey: 'machines.machineCreateWizard.location.local.desc',
    },
    {
      value: 'remote',
      icon: ArrowRight,
      titleKey: 'machines.machineCreateWizard.location.remote.title',
      descKey: 'machines.machineCreateWizard.location.remote.desc',
    },
  ]

export const machineWizardStrategyOptions: MachineWizardOptionCard<MachineWizardStrategy>[] = [
  {
    value: 'ssh-install-listener',
    icon: Sparkles,
    titleKey: 'machines.machineCreateWizard.strategy.sshInstall.title',
    descKey: 'machines.machineCreateWizard.strategy.sshInstall.desc',
  },
  {
    value: 'reverse',
    icon: ArrowLeftRight,
    titleKey: 'machines.machineCreateWizard.strategy.reverse.title',
    descKey: 'machines.machineCreateWizard.strategy.reverse.desc',
  },
  {
    value: 'direct-open',
    icon: Radio,
    titleKey: 'machines.machineCreateWizard.strategy.directOpen.title',
    descKey: 'machines.machineCreateWizard.strategy.directOpen.desc',
  },
]
