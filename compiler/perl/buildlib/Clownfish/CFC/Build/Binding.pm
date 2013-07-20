# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

package Clownfish::CFC::Build::Binding;
use strict;

sub bind_all {
    my $class = shift;
    $class->bind_perlclass;
    $class->bind_perlpod;
}

sub bind_perlclass {
    class_from_c('CFCPerlClass', 'Clownfish::CFC::Binding::Perl::Class');

    my @exposed = qw(
        Add_To_Registry
        Bind_Constructor
        Bind_Method
        Exclude_Constructor
        Exclude_Method
        Create_Pod
        Get_Client
        Get_Class_Name
        Append_Xs
        Get_Xs_Code
        Set_Pod_Spec
        Get_Pod_Spec
    );
    # TODO: Generate docs for functions
    #my @exposed_functions = qw(
    #    singleton
    #    registry
    #    clear_registry
    #    method_bindings
    #    constructor_bindings
    #);

    my $pod_spec = Clownfish::CFC::Binding::Perl::Pod->new;
    $pod_spec->add_constructor( alias => 'new' );
    $pod_spec->add_method( method => $_, alias => lc($_) ) for @exposed;

    my $binding = Clownfish::CFC::Binding::Perl::Class->new(
        parcel     => 'Clownfish',
        class_name => 'Clownfish::CFC::Binding::Perl::Class',
    );
    $binding->set_pod_spec($pod_spec);

    Clownfish::CFC::Binding::Perl::Class->register($binding);
}

sub bind_perlpod {
    class_from_c('CFCPerlPod', 'Clownfish::CFC::Binding::Perl::Pod');

    my @exposed = qw(
        Add_Method
        Add_Constructor
        Set_Synopsis
        Get_Synopsis
        Set_Description
        Get_Description
    );

    my $pod_spec = Clownfish::CFC::Binding::Perl::Pod->new;
    $pod_spec->add_constructor( alias => 'new' );
    $pod_spec->add_method( method => $_, alias => lc($_) ) for @exposed;

    my $binding = Clownfish::CFC::Binding::Perl::Class->new(
        parcel     => 'Clownfish',
        class_name => 'Clownfish::CFC::Binding::Perl::Pod',
    );
    $binding->set_pod_spec($pod_spec);

    Clownfish::CFC::Binding::Perl::Class->register($binding);
}

# Quick and dirty hack to create a CFC class from a C header.
sub class_from_c {
    my ($cfc_class_name, $class_name) = @_;

    $class_name =~ /::(\w+)$/;
    my $last_component = $1;

    my $h_filename = "../src/$cfc_class_name.h";
    open(my $h_file, '<', $h_filename)
        or die("$h_filename: $!");

    my ($class_doc, $function_doc, @functions);
    my $state = 'before_class_doc';

    while (my $line = <$h_file>) {
        if ($state eq 'before_class_doc') {
            if ($line =~ m"^/\*\*") {
                $class_doc = $line;
                $state = 'in_class_doc';
            }
        }
        elsif ($state eq 'in_class_doc') {
            $class_doc .= $line;
            if ($line =~ m"\*/") {
                $state = 'before_function_doc';
            }
        }
        elsif ($state eq 'before_function_doc') {
            if ($line =~ m"^/\*\*") {
                $function_doc = $line;
                $state = 'in_function_doc';
            }
        }
        elsif ($state eq 'in_function_doc') {
            $function_doc .= $line;
            if ($line =~ m"\*/") {
                $state = 'after_function_doc';
            }
        }
        elsif ($state eq 'after_function_doc') {
            if ($line =~ m"^${cfc_class_name}_(\w+)(\(.*)") {
                my $function_name = $1;
                my $c_params      = $2;

                while ($c_params !~ /\)/) {
                    $c_params .= <$h_file>;
                }

                $c_params =~ s/\).*/\)/;

                push(@functions, {
                    name     => $function_name,
                    c_params => $c_params,
                    doc      => $function_doc,
                });

                $state = 'before_function_doc';
            }
        }
    }

    close($h_file);

    my $class = Clownfish::CFC::Model::Class->create(
        parcel      => 'Clownfish',
        class_name  => $class_name,
        docucomment => Clownfish::CFC::Model::DocuComment->parse($class_doc),
    );

    my $parser    = Clownfish::CFC::Parser->new;
    my $void_type = Clownfish::CFC::Model::Type->new_void;

    $parser->parse('parcel Clownfish;');

    for my $function (@functions) {
        my $name     = $function->{name};
        my $c_params = $function->{c_params};
        my $dc = Clownfish::CFC::Model::DocuComment->parse($function->{doc});

        # "struct" is not allowed in Clownfish types.
        $c_params =~ s/\bstruct //g;
        # Clownfish reserved words.
        $c_params =~ s/\bparcel\b/_$&/g;
        # Empty param list.
        $c_params = '()' if $c_params eq '(void)';

        if ($c_params =~ /\($cfc_class_name \*self/) {
            my $macro_sym = $name;
            $macro_sym =~ s/(^|_)([a-z])/$1 . uc($2)/ge;

            $c_params =~ s/^\(\w+/($last_component/;
            my $param_list = $parser->parse($c_params)
                or die("Invalid param list: $c_params");

            my $method = Clownfish::CFC::Model::Method->new(
                parcel      => 'Clownfish',
                class_name  => $class_name,
                exposure    => 'public',
                macro_sym   => $macro_sym,
                docucomment => $dc,
                return_type => $void_type,
                param_list  => $param_list,
            );
            $class->add_method($method);
        }
        else {
            $name = 'init' if $name eq 'new';

            my $param_list = $parser->parse($c_params)
                or die("Invalid param list: $c_params");

            my $function = Clownfish::CFC::Model::Function->new(
                parcel      => 'Clownfish',
                class_name  => $class_name,
                exposure    => 'public',
                micro_sym   => $name,
                docucomment => $dc,
                return_type => $void_type,
                param_list  => $param_list,
            );
            $class->add_function($function);
        }
    }

    return $class;
}

1;

