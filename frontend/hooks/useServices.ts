import { useState, useEffect } from 'react';

export interface ConfigField {
  name: string;
  label: string;
  type: string;
  required: boolean;
  description: string;
  default?: string;
}

export interface Service {
  name: string;
  slug: string;
  description: string;
  icon: string;
  is_enabled: boolean;
  schema: ConfigField[];
}

export function useServices() {
  const [services, setServices] = useState<Service[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const fetchServices = async () => {
    try {
      const response = await fetch('/api/services');
      if (!response.ok) throw new Error('Failed to fetch services');
      const data = await response.json();
      setServices(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchServices();
  }, []);

  return { services, loading, error, refresh: fetchServices };
}
