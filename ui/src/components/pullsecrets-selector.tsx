import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { listNamespacedPullSecrets } from "@/lib/kubernetes";
import { useQuery } from "@tanstack/react-query";
import { ChevronDown, TriangleAlert } from "lucide-react";
import type { ReactElement } from "react";

interface PullSecretsSelectorProps {
  readonly namespace: string;
  readonly pullSecrets: string[];
  readonly placeholder: string;
  readonly setPullSecrets: (secrets: string[]) => void;
}

function renderDropdownMenuTrigger(
  data: string[],
  secrets: string[],
  namespace: string,
  placeholder: string,
): ReactElement {
  if (data.length === 0) {
    return (
      <>
        <span>No pull secrets in namespace {namespace}</span>
        <TriangleAlert className="ml-2 h-4 w-4 fill-red-500 opacity-50" />
      </>
    );
  }

  if (secrets.length > 0) {
    return <span>{secrets.join(", ")}</span>;
  }

  return (
    <>
      <span>{placeholder}</span>
    </>
  );
}

export function PullSecretsSelector(props: PullSecretsSelectorProps): ReactElement {
  const { namespace, placeholder, pullSecrets, setPullSecrets } = props;

  const { data, isPending, isError, error } = useQuery<string[], Error>({
    queryFn: async () => {
      const response = await listNamespacedPullSecrets(namespace);

      return JSON.parse(response);
    },
    queryKey: ["pullsecrets", { namespace: namespace }],
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
            {renderDropdownMenuTrigger(data, pullSecrets, namespace, placeholder)}
            <ChevronDown className="ml-2 h-4 w-4 opacity-50" />
          </Button>
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuContent style={{ minWidth: "30vw" }}>
        {data.map((curSecret: string) => {
          return (
            <DropdownMenuCheckboxItem
              checked={pullSecrets.includes(curSecret)}
              key={curSecret}
              disabled={data.length === 0}
              onClick={() => {
                const clonedSecrets = [...pullSecrets];

                if (pullSecrets.includes(curSecret)) {
                  setPullSecrets(
                    clonedSecrets.filter((element) => {
                      return element !== curSecret;
                    }),
                  );
                  return;
                }

                clonedSecrets.push(curSecret);
                setPullSecrets(clonedSecrets);
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
