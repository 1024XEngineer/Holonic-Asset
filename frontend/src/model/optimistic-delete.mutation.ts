import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { MutationKey } from "@tanstack/react-query";

type OptimisticDeleteOptions<TData, TInput, TSnapshot> = {
  mutationKey: MutationKey;
  mutationFn: (input: TInput) => Promise<TData>;
  removeFromCache: (input: TInput) => Promise<TSnapshot> | TSnapshot;
  restoreCache: (input: TInput, snapshot: TSnapshot) => void;
  refreshCache: (input: TInput) => void;
  isSameScope: (variables: TInput | undefined, input: TInput) => boolean;
  onSuccess?: (data: TData, input: TInput) => void;
};

export function useOptimisticDeleteMutation<TData, TInput, TSnapshot>(
  options: OptimisticDeleteOptions<TData, TInput, TSnapshot>,
) {
  const queryClient = useQueryClient();

  return useMutation<TData, Error, TInput, TSnapshot>({
    mutationKey: options.mutationKey,
    mutationFn: options.mutationFn,
    onMutate: options.removeFromCache,
    onSuccess: options.onSuccess,
    onError: (_error, input, snapshot) => {
      if (snapshot === undefined) return;
      options.restoreCache(input, snapshot);
    },
    onSettled: (_data, _error, input) => {
      if (
        queryClient.isMutating({
          mutationKey: options.mutationKey,
          predicate: (mutation) =>
            options.isSameScope(
              mutation.state.variables as TInput | undefined,
              input,
            ),
        }) !== 1
      ) {
        return;
      }
      options.refreshCache(input);
    },
  });
}
