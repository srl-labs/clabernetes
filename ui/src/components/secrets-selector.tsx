import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { listNamespacedSecrets } from "@/lib/kubernetes";
import { useQuery } from "@tanstack/react-query";
import { ChevronDown, TriangleAlert } from "lucide-react";
import type { ReactElement } from "react";

interface SecretSelectorProps {
  readonly namespace: string;
  readonly secret: string;
  readonly placeholder: string;
  readonly setSecret: (secret: string) => void;
}

function renderDropdownMenuTrigger(
  data: string[],
  secret: string,
  namespace: string,
  placeholder: string,
): ReactElement {
  if (data.length === 0) {
    return (
      <>
        <span>No secrets in namespace {namespace}</span>
        <TriangleAlert className="ml-2 h-4 w-4 fill-red-500 opacity-50" />
      </>
    );
  }

  if (secret !== "") {
    return <span>{secret}</span>;
  }

  return (
    <>
      <span>{placeholder}</span>
    </>
  );
}

export function SecretSelector(props: SecretSelectorProps): ReactElement {
  const { namespace, placeholder, secret, setSecret } = props;

  const { data, isPending, isError, error } = useQuery<string[], Error>({
    queryFn: async () => {
      const response = await listNamespacedSecrets(namespace);

      return JSON.parse(response);
    },
    queryKey: ["secrets", { namespace: namespace }],
  });

  if (isPending) {
    return (
      <div className="flex justify-end">
        <Button
          className="flex w-full justify-between"
          variant="outline"
        >
          <span>Loading...</span>
          <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
        </Button>
      </div>
    );
  }

  if (isError) {
    return <span>Error: {error.message}</span>;
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        asChild={true}
        className="col-span-3"
      >
        <div className="flex justify-end">
          <Button
            className="flex w-full justify-between"
            variant="outline"
          >
            {renderDropdownMenuTrigger(data, secret, namespace, placeholder)}
            <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
          </Button>
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuContent style={{ minWidth: "30vw" }}>
        {data.map((curSecret: string) => {
          return (
            <DropdownMenuCheckboxItem
              checked={secret === curSecret}
              key={curSecret}
              disabled={data.length === 0}
              onClick={() => {
                setSecret(curSecret);
              }}
            >
              {curSecret}
            </DropdownMenuCheckboxItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
