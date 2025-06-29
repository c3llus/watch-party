export interface DeploymentConfig {
  mode: 'saas' | 'self-hosted'
  cdn_enabled: boolean
  storage_type: 'gcs' | 'minio'
  features: {
    tunneling_available: boolean
  }
}

class ConfigService {
  private config: DeploymentConfig | null = null

  // get deployment configuration
  async getDeploymentConfig(): Promise<DeploymentConfig> {
    if (this.config) {
      return this.config
    }

    // force proxy mode for development to test per-request auth
    const isSelfHosted = false  // change this to true to test self-hosted mode
    
    this.config = {
      mode: isSelfHosted ? 'self-hosted' : 'saas',
      cdn_enabled: !isSelfHosted,
      storage_type: isSelfHosted ? 'minio' : 'gcs',
      features: {
        tunneling_available: isSelfHosted
      }
    }

    return this.config
  }

  // check if CDN is enabled
  async isCDNEnabled(): Promise<boolean> {
    const config = await this.getDeploymentConfig()
    return config.cdn_enabled
  }

  // check if self-hosted mode
  async isSelfHosted(): Promise<boolean> {
    const config = await this.getDeploymentConfig()
    return config.mode === 'self-hosted'
  }

  // clear cached config (useful for testing different modes)
  clearCache(): void {
    this.config = null
  }
}

export const configService = new ConfigService()
