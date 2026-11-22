import { keepPreviousData, useQuery } from "@tanstack/react-query";

import { userSearchQuery } from "../../../api/queries";
import { useDebouncedValue } from "../../../lib/useDebouncedValue";

const SEARCH_DELAY_MS = 250;

export function usePeopleSearch(query: string) {
  const debounced = useDebouncedValue(query.trim(), SEARCH_DELAY_MS);
  const result = useQuery({ ...userSearchQuery(debounced), placeholderData: keepPreviousData });
  return { ...result, isContacts: debounced === "" };
}
