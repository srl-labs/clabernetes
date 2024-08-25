import { type ReactElement, useState } from "react";
import {
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet.tsx";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
} from "@/components/ui/form.tsx";
import { Input } from "@/components/ui/input.tsx";
import { NamespaceSelector } from "@/components/namespace-selector.tsx";
import { Button } from "@/components/ui/button.tsx";
import { getQueryClient } from "@/lib/get-query-client.ts";
import type { ClabernetesContainerlabDevTopologyV1Alpha1 } from "@/lib/clabernetes-client";
import { createTopology } from "@/lib/kubernetes.ts";
import { Textarea } from "@/components/ui/textarea.tsx";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog.tsx";

interface CreateAlertDialogProps {
  readonly title: string;
  readonly description: string;
}

function AddAlert(
  alert: CreateAlertDialogProps,
  alerts: CreateAlertDialogProps[],
  setAlerts: (alerts: CreateAlertDialogProps[]) => void,
) {
  if (alerts.length === 0) {
    setAlerts([alert]);

    return;
  }

  const clonedAlerts = [...alerts];

  if (alerts.includes(alert)) {
    setAlerts(
      clonedAlerts.filter((element) => {
        return element !== alert;
      }),
    );
    return;
  }

  clonedAlerts.push(alert);
  setAlerts(clonedAlerts);
}

export const formSchema = z.object({
  name: z.string().min(2),
  namespace: z.string(),
  definition: z.string(),
});

const formDefaultValues = {
  name: "",
  namespace: "default",
  definition: "",
};

export function CreateSheet(): ReactElement {
  const [alerts, setAlerts] = useState<CreateAlertDialogProps[]>([]);

  const form = useForm<z.infer<typeof formSchema>>({
    defaultValues: formDefaultValues,
    resolver: zodResolver(formSchema),
  });

  async function onSubmitWrapper(values: z.infer<typeof formSchema>): Promise<void> {
    const obj = {
      apiVersion: "clabernetes.containerlab.dev/v1alpha1",
      kind: "Topology",
      metadata: {
        name: values.name,
        namespace: values.namespace,
      },
      spec: {
        definition: {
          containerlab: values.definition,
        },
        naming: "global",
      },
    } as ClabernetesContainerlabDevTopologyV1Alpha1;

    const response = await createTopology(values.namespace, values.name, JSON.stringify(obj));

    const parsedResponse = JSON.parse(response);

    if (parsedResponse.error) {
      AddAlert(
        {
          title: "Create Topology Failed! :(",
          description: `Error: ${parsedResponse.error.message}`,
        },
        alerts,
        setAlerts,
      );

      return;
    }

    form.reset();

    getQueryClient()
      .invalidateQueries({
        queryKey: ["topologies", {}],
      })
      .catch((invalidateError: unknown) => {
        throw invalidateError;
      });
  }

  return (
    <>
      {alerts.map((alert) => {
        return (
          <AlertDialog
            key={"alert"}
            onOpenChange={() => {
              const clonedAlerts = [...alerts];

              if (alerts.includes(alert)) {
                setAlerts(
                  clonedAlerts.filter((element) => {
                    return element !== alert;
                  }),
                );
                return;
              }

              clonedAlerts.push(alert);
              setAlerts(clonedAlerts);
            }}
            open={alerts.length > 0}
          >
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{alert.title}</AlertDialogTitle>
                <AlertDialogDescription>{alert.description}</AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Ok</AlertDialogCancel>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        );
      })}
      <SheetContent
        className="max-h-screen overflow-y-auto"
        style={{ maxWidth: "50vw" }}
        onSubmit={form.handleSubmit(onSubmitWrapper)}
      >
        <Form {...form}>
          <form className="space-y-8">
            <SheetHeader>
              <SheetTitle>Create a Topology</SheetTitle>
              <SheetDescription>
                A Topology holds a valid containerlab topology and runs it in Kubernetes
              </SheetDescription>
            </SheetHeader>
            <div className="grid gap-4 py-4">
              <FormField
                control={form.control}
                name={"name"}
                render={({ field: { onChange, value } }) => {
                  return (
                    <FormItem>
                      <div className="grid grid-cols-4 items-center gap-4">
                        <FormLabel
                          className="text-right"
                          htmlFor="name"
                        >
                          Name
                        </FormLabel>
                        <FormControl>
                          <Input
                            className="col-span-3"
                            onChange={onChange}
                            value={value}
                          />
                        </FormControl>
                      </div>
                      <FormDescription>The name of the clabernetes Topology</FormDescription>
                    </FormItem>
                  );
                }}
              />
              <FormField
                control={form.control}
                name={"namespace"}
                render={({ field: { onChange, value } }) => {
                  return (
                    <FormItem>
                      <div className="grid grid-cols-4 items-center gap-4">
                        <FormLabel
                          className="text-right"
                          htmlFor="namespace"
                        >
                          Namespace
                        </FormLabel>
                        <FormControl>
                          <NamespaceSelector
                            namespace={value}
                            placeholder=""
                            setNamespace={onChange}
                          />
                        </FormControl>
                      </div>
                      <FormDescription>The namespace to create the Topology in</FormDescription>
                    </FormItem>
                  );
                }}
              />
              <FormField
                control={form.control}
                name={"definition"}
                render={({ field: { onChange, value } }) => {
                  return (
                    <FormItem>
                      <div className="grid grid-cols-4 items-center gap-4">
                        <FormLabel
                          className="text-right"
                          htmlFor="definition"
                        >
                          Definition
                        </FormLabel>
                        <FormControl>
                          <Textarea
                            className="col-span-3"
                            onChange={onChange}
                            value={value}
                          />
                        </FormControl>
                      </div>
                      <FormDescription>
                        The containerlab topology to run in Kubernetes
                      </FormDescription>
                    </FormItem>
                  );
                }}
              />
            </div>
            <SheetFooter>
              <Button
                onClick={() => {
                  form.reset();
                }}
                type="button"
                variant="destructive"
              >
                Reset
              </Button>
              <SheetClose asChild={true}>
                <Button type="submit">Submit</Button>
              </SheetClose>
            </SheetFooter>
          </form>
        </Form>
      </SheetContent>
    </>
  );
}
