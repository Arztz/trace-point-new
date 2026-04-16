import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../utils/api';

export function useDatasources() {
  return useQuery({
    queryKey: ['datasources'],
    queryFn: () => api.getDatasources(),
    staleTime: Infinity,
  });
}

export function useTimeline(timeRange, deploymentName) {
  return useQuery({
    queryKey: ['timeline', timeRange, deploymentName],
    queryFn: () => api.getTimeline(timeRange, deploymentName),
    staleTime: Infinity, // No auto-refresh, manual only
  });
}

export function useSpikes(params = {}) {
  return useQuery({
    queryKey: ['spikes', params],
    queryFn: () => api.getSpikes(params),
    staleTime: 10000,
  });
}

export function useSpikeDetails(id) {
  return useQuery({
    queryKey: ['spike', id],
    queryFn: () => api.getSpikeDetails(id),
    enabled: !!id,
  });
}

export function useRetrySpike() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (id) => api.retrySpikeCorrelation(id),
    onSuccess: (data, id) => {
      // Invalidate both lists and detail views to refresh data
      queryClient.invalidateQueries({ queryKey: ['spikes'] });
      queryClient.invalidateQueries({ queryKey: ['spike', id] });
    },
  });
}

export function useAnalyzeSpikes(params, enabled = true) {
  return useQuery({
    queryKey: ['analyze', params],
    queryFn: () => api.analyzeSpikes(params),
    enabled,
    staleTime: 60000,
  });
}

export function useGravityScores() {
  return useQuery({
    queryKey: ['gravity-scores'],
    queryFn: () => api.getGravityScores(),
    staleTime: 300000, // 5 minutes
  });
}

export function useConfig() {
  return useQuery({
    queryKey: ['config'],
    queryFn: () => api.getConfig(),
    staleTime: 600000, // 10 minutes
  });
}
