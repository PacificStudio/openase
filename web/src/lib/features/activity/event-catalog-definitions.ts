import { activityEventCoreDefinitions } from './event-catalog-definitions-core'
import { activityEventIntegrationDefinitions } from './event-catalog-definitions-integrations'
import { activityEventRuntimeDefinitions } from './event-catalog-definitions-runtime'
import type { ActivityEventCatalogDefinition } from './event-catalog-types'

export const activityEventDefinitions: ActivityEventCatalogDefinition[] = [
  ...activityEventCoreDefinitions,
  ...activityEventRuntimeDefinitions,
  ...activityEventIntegrationDefinitions,
]
