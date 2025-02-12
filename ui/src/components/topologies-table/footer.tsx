import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { getQueryClient } from "@/lib/get-query-client";
import { RefreshCw } from "lucide-react";
import type { ReactElement } from "react";
import { RefetchInterval } from "@/components/topologies-table/table.tsx";

interface FooterProps {
  readonly getCanNextPage: () => boolean;
  readonly getCanPreviousPage: () => boolean;
  readonly nextPage: () => void;
  readonly previousPage: () => void;
  readonly refetch: string;
  readonly setRefetch: (refetch: RefetchInterval) => void;
}

export function Footer(props: FooterProps): ReactElement {
  const { getCanNextPage, getCanPreviousPage, nextPage, previousPage, refetch, setRefetch } = props;

  return (
    <div className="flex justify-between">
      <div className="flex space-x-2 py-4">
        <Button
          onClick={(): void => {
            getQueryClient()
              .invalidateQueries({
                queryKey: ["topologies", {}],
              })
              .catch((invalidateError: unknown) => {
                throw invalidateError;
              });
          }}
          size="sm"
          variant="outline"
        >
          <RefreshCw className="h-4 w-4" />
        </Button>
        <DropdownMenu>
          <DropdownMenuTrigger asChild={true}>
            <Button
              size="sm"
              variant="outline"
            >
              Refresh: {refetch}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuRadioGroup
              onValueChange={(value: string): void => {
                switch (value) {
                  case "3s":
                    setRefetch(RefetchInterval.ThreeSeconds);
                    break;
                  case "10s":
                    setRefetch(RefetchInterval.TenSeconds);
                    break;
                  case "60s":
                    setRefetch(RefetchInterval.OneMinute);
                    break;
                  case "300s":
                    setRefetch(RefetchInterval.FiveMinutes);
                    break;
                  default:
                    setRefetch(RefetchInterval.Never);
                }
              }}
              value={refetch}
            >
              <DropdownMenuRadioItem value={RefetchInterval.ThreeSeconds}>3s</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value={RefetchInterval.TenSeconds}>10s</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value={RefetchInterval.OneMinute}>1m</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value={RefetchInterval.FiveMinutes}>5m</DropdownMenuRadioItem>
              <DropdownMenuRadioItem value={RefetchInterval.Never}>Never</DropdownMenuRadioItem>
            </DropdownMenuRadioGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="ml-auto flex space-x-2 py-4">
        <Button
          disabled={!getCanPreviousPage()}
          onClick={(): void => {
            previousPage();
          }}
          size="sm"
          variant="outline"
        >
          Previous
        </Button>
        <Button
          disabled={!getCanNextPage()}
          onClick={(): void => {
            nextPage();
          }}
          size="sm"
          variant="outline"
        >
          Next
        </Button>
      </div>
    </div>
  );
}
