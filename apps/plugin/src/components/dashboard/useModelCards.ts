import { useState, useEffect, useMemo } from 'react';
import type { DashboardDataSource } from '../../dashboard/api';
import { type PricingMap, buildPricingMap } from '../../dashboard/cost';
import type { ModelCard } from '../../dashboard/types';

type ModelCardsResult = {
  cards: ModelCard[];
  pricingMap: PricingMap;
  loading: boolean;
  error: string;
};

export function useModelCards(dataSource: DashboardDataSource): ModelCardsResult {
  const [cards, setCards] = useState<ModelCard[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let cancelled = false;

    const run = async () => {
      try {
        const result = await dataSource.listModelCards();
        if (!cancelled) {
          setCards(result);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : 'Failed to load model cards');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    };

    run();
    return () => {
      cancelled = true;
    };
  }, [dataSource]);

  const pricingMap = useMemo(() => buildPricingMap(cards), [cards]);

  return { cards, pricingMap, loading, error };
}
