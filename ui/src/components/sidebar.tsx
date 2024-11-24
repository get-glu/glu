import { Button } from '@/components/ui/button';
import stu from '@/assets/stu.png';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem
} from '@/components/ui/command';
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover';
import { cn } from '@/lib/utils';
import { useState } from 'react';
import {
  Sidebar as SidebarComponent,
  SidebarContent,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuItem
} from '@/components/ui/sidebar';
import { Check, ChevronsUpDown, BookOpen, Github } from 'lucide-react';
import { Link, useNavigate } from 'react-router-dom';
import { Pipeline } from '@/types/pipeline';
import { useListPipelinesQuery } from '@/services/api';
import { useParams } from 'react-router-dom';

export function Sidebar() {
  const navigate = useNavigate();
  const { pipelineId } = useParams();
  const [open, setOpen] = useState(false);
  const { data: pipelinesData, isLoading } = useListPipelinesQuery();

  return (
    <SidebarComponent>
      <SidebarContent className="flex h-full flex-col justify-between">
        <SidebarGroup>
          <SidebarGroupLabel className="mb-2">
            <Link to="/">
              <div className="flex items-center gap-2">
                <img src={stu} alt="Stu" className="h-8 w-8" />
                <h1 className="text-lg font-bold">Glu</h1>
              </div>
            </Link>
          </SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <Popover open={open} onOpenChange={setOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      variant="outline"
                      role="combobox"
                      aria-expanded={open}
                      className="mt-2 w-full justify-between"
                      disabled={isLoading}
                    >
                      {isLoading
                        ? 'Loading pipelines...'
                        : pipelineId
                          ? pipelinesData?.pipelines?.find(
                              (pipeline: Pipeline) => pipeline.name === pipelineId
                            )?.name
                          : 'Select pipeline...'}
                      <ChevronsUpDown className="ml-2 h-4 w-4 shrink-0 opacity-50" />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-fit p-0">
                    <Command>
                      <CommandInput placeholder="Search pipelines..." />
                      <CommandEmpty>No pipeline found.</CommandEmpty>
                      <CommandGroup>
                        {isLoading ? (
                          <CommandItem disabled>Loading pipelines...</CommandItem>
                        ) : (
                          pipelinesData?.pipelines?.map((pipeline: Pipeline) => (
                            <CommandItem
                              key={pipeline.name}
                              value={pipeline.name}
                              onSelect={(currentValue: string) => {
                                setOpen(false);
                                navigate(`/pipelines/${currentValue}`);
                              }}
                              className="truncate"
                            >
                              <Check
                                className={cn(
                                  'mr-2 h-4 w-4',
                                  pipelineId === pipeline.name ? 'opacity-100' : 'opacity-0'
                                )}
                              />
                              {pipeline.name}
                            </CommandItem>
                          ))
                        )}
                      </CommandGroup>
                    </Command>
                  </PopoverContent>
                </Popover>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>

        <SidebarGroup className="mt-auto">
          <SidebarGroupContent>
            <SidebarMenu>
              <SidebarMenuItem>
                <Button
                  variant="ghost"
                  className="w-full justify-start"
                  onClick={() => window.open('https://docs.getglu.dev', '_blank')}
                >
                  <BookOpen className="mr-2 h-4 w-4" />
                  Documentation
                </Button>
              </SidebarMenuItem>
              <SidebarMenuItem>
                <Button
                  variant="ghost"
                  className="w-full justify-start"
                  onClick={() => window.open('https://github.com/get-glu/glu', '_blank')}
                >
                  <Github className="mr-2 h-4 w-4" />
                  GitHub
                </Button>
              </SidebarMenuItem>
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </SidebarComponent>
  );
}
