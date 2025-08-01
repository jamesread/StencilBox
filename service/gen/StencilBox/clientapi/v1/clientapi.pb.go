// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.36.6
// 	protoc        (unknown)
// source: StencilBox/clientapi/v1/clientapi.proto

package clientapi_pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
	unsafe "unsafe"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type InitRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *InitRequest) Reset() {
	*x = InitRequest{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[0]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *InitRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InitRequest) ProtoMessage() {}

func (x *InitRequest) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[0]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InitRequest.ProtoReflect.Descriptor instead.
func (*InitRequest) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{0}
}

type InitResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Version       string                 `protobuf:"bytes,1,opt,name=version,proto3" json:"version,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *InitResponse) Reset() {
	*x = InitResponse{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[1]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *InitResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*InitResponse) ProtoMessage() {}

func (x *InitResponse) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[1]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use InitResponse.ProtoReflect.Descriptor instead.
func (*InitResponse) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{1}
}

func (x *InitResponse) GetVersion() string {
	if x != nil {
		return x.Version
	}
	return ""
}

type Template struct {
	state            protoimpl.MessageState `protogen:"open.v1"`
	Name             string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Source           string                 `protobuf:"bytes,2,opt,name=source,proto3" json:"source,omitempty"`
	Status           string                 `protobuf:"bytes,3,opt,name=status,proto3" json:"status,omitempty"`
	DocumentationUrl string                 `protobuf:"bytes,4,opt,name=documentation_url,json=documentationUrl,proto3" json:"documentation_url,omitempty"`
	BuildConfigs     []string               `protobuf:"bytes,5,rep,name=build_configs,json=buildConfigs,proto3" json:"build_configs,omitempty"`
	unknownFields    protoimpl.UnknownFields
	sizeCache        protoimpl.SizeCache
}

func (x *Template) Reset() {
	*x = Template{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[2]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *Template) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Template) ProtoMessage() {}

func (x *Template) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[2]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Template.ProtoReflect.Descriptor instead.
func (*Template) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{2}
}

func (x *Template) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Template) GetSource() string {
	if x != nil {
		return x.Source
	}
	return ""
}

func (x *Template) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *Template) GetDocumentationUrl() string {
	if x != nil {
		return x.DocumentationUrl
	}
	return ""
}

func (x *Template) GetBuildConfigs() []string {
	if x != nil {
		return x.BuildConfigs
	}
	return nil
}

type BuildConfig struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Name          string                 `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Template      string                 `protobuf:"bytes,2,opt,name=template,proto3" json:"template,omitempty"`
	OutputDir     string                 `protobuf:"bytes,3,opt,name=output_dir,json=outputDir,proto3" json:"output_dir,omitempty"`
	Repos         []string               `protobuf:"bytes,4,rep,name=repos,proto3" json:"repos,omitempty"`
	Datafiles     map[string]string      `protobuf:"bytes,5,rep,name=datafiles,proto3" json:"datafiles,omitempty" protobuf_key:"bytes,1,opt,name=key" protobuf_val:"bytes,2,opt,name=value"`
	Filename      string                 `protobuf:"bytes,6,opt,name=filename,proto3" json:"filename,omitempty"`
	Path          string                 `protobuf:"bytes,7,opt,name=path,proto3" json:"path,omitempty"`
	ErrorMessage  string                 `protobuf:"bytes,8,opt,name=error_message,json=errorMessage,proto3" json:"error_message,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BuildConfig) Reset() {
	*x = BuildConfig{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[3]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BuildConfig) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildConfig) ProtoMessage() {}

func (x *BuildConfig) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[3]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildConfig.ProtoReflect.Descriptor instead.
func (*BuildConfig) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{3}
}

func (x *BuildConfig) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *BuildConfig) GetTemplate() string {
	if x != nil {
		return x.Template
	}
	return ""
}

func (x *BuildConfig) GetOutputDir() string {
	if x != nil {
		return x.OutputDir
	}
	return ""
}

func (x *BuildConfig) GetRepos() []string {
	if x != nil {
		return x.Repos
	}
	return nil
}

func (x *BuildConfig) GetDatafiles() map[string]string {
	if x != nil {
		return x.Datafiles
	}
	return nil
}

func (x *BuildConfig) GetFilename() string {
	if x != nil {
		return x.Filename
	}
	return ""
}

func (x *BuildConfig) GetPath() string {
	if x != nil {
		return x.Path
	}
	return ""
}

func (x *BuildConfig) GetErrorMessage() string {
	if x != nil {
		return x.ErrorMessage
	}
	return ""
}

type BuildRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ConfigName    string                 `protobuf:"bytes,1,opt,name=config_name,json=configName,proto3" json:"config_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BuildRequest) Reset() {
	*x = BuildRequest{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[4]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BuildRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildRequest) ProtoMessage() {}

func (x *BuildRequest) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[4]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildRequest.ProtoReflect.Descriptor instead.
func (*BuildRequest) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{4}
}

func (x *BuildRequest) GetConfigName() string {
	if x != nil {
		return x.ConfigName
	}
	return ""
}

type BuildResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ConfigName    string                 `protobuf:"bytes,1,opt,name=config_name,json=configName,proto3" json:"config_name,omitempty"`
	Status        string                 `protobuf:"bytes,2,opt,name=status,proto3" json:"status,omitempty"`
	Found         bool                   `protobuf:"varint,3,opt,name=found,proto3" json:"found,omitempty"`
	RelativePath  string                 `protobuf:"bytes,4,opt,name=relative_path,json=relativePath,proto3" json:"relative_path,omitempty"`
	IsError       bool                   `protobuf:"varint,5,opt,name=isError,proto3" json:"isError,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *BuildResponse) Reset() {
	*x = BuildResponse{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[5]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *BuildResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*BuildResponse) ProtoMessage() {}

func (x *BuildResponse) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[5]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use BuildResponse.ProtoReflect.Descriptor instead.
func (*BuildResponse) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{5}
}

func (x *BuildResponse) GetConfigName() string {
	if x != nil {
		return x.ConfigName
	}
	return ""
}

func (x *BuildResponse) GetStatus() string {
	if x != nil {
		return x.Status
	}
	return ""
}

func (x *BuildResponse) GetFound() bool {
	if x != nil {
		return x.Found
	}
	return false
}

func (x *BuildResponse) GetRelativePath() string {
	if x != nil {
		return x.RelativePath
	}
	return ""
}

func (x *BuildResponse) GetIsError() bool {
	if x != nil {
		return x.IsError
	}
	return false
}

type GetTemplatesRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetTemplatesRequest) Reset() {
	*x = GetTemplatesRequest{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[6]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetTemplatesRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetTemplatesRequest) ProtoMessage() {}

func (x *GetTemplatesRequest) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[6]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetTemplatesRequest.ProtoReflect.Descriptor instead.
func (*GetTemplatesRequest) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{6}
}

type GetTemplatesResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Templates     []*Template            `protobuf:"bytes,1,rep,name=templates,proto3" json:"templates,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetTemplatesResponse) Reset() {
	*x = GetTemplatesResponse{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[7]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetTemplatesResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetTemplatesResponse) ProtoMessage() {}

func (x *GetTemplatesResponse) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[7]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetTemplatesResponse.ProtoReflect.Descriptor instead.
func (*GetTemplatesResponse) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{7}
}

func (x *GetTemplatesResponse) GetTemplates() []*Template {
	if x != nil {
		return x.Templates
	}
	return nil
}

type GetStatusRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetStatusRequest) Reset() {
	*x = GetStatusRequest{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[8]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetStatusRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetStatusRequest) ProtoMessage() {}

func (x *GetStatusRequest) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[8]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetStatusRequest.ProtoReflect.Descriptor instead.
func (*GetStatusRequest) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{8}
}

type GetStatusResponse struct {
	state           protoimpl.MessageState `protogen:"open.v1"`
	InContainer     bool                   `protobuf:"varint,2,opt,name=in_container,json=inContainer,proto3" json:"in_container,omitempty"`
	TemplatesPath   string                 `protobuf:"bytes,4,opt,name=templates_path,json=templatesPath,proto3" json:"templates_path,omitempty"`
	OutputPath      string                 `protobuf:"bytes,5,opt,name=output_path,json=outputPath,proto3" json:"output_path,omitempty"`
	BuildConfigsDir string                 `protobuf:"bytes,6,opt,name=build_configs_dir,json=buildConfigsDir,proto3" json:"build_configs_dir,omitempty"`
	unknownFields   protoimpl.UnknownFields
	sizeCache       protoimpl.SizeCache
}

func (x *GetStatusResponse) Reset() {
	*x = GetStatusResponse{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[9]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetStatusResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetStatusResponse) ProtoMessage() {}

func (x *GetStatusResponse) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[9]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetStatusResponse.ProtoReflect.Descriptor instead.
func (*GetStatusResponse) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{9}
}

func (x *GetStatusResponse) GetInContainer() bool {
	if x != nil {
		return x.InContainer
	}
	return false
}

func (x *GetStatusResponse) GetTemplatesPath() string {
	if x != nil {
		return x.TemplatesPath
	}
	return ""
}

func (x *GetStatusResponse) GetOutputPath() string {
	if x != nil {
		return x.OutputPath
	}
	return ""
}

func (x *GetStatusResponse) GetBuildConfigsDir() string {
	if x != nil {
		return x.BuildConfigsDir
	}
	return ""
}

type GetBuildConfigsRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetBuildConfigsRequest) Reset() {
	*x = GetBuildConfigsRequest{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[10]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetBuildConfigsRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBuildConfigsRequest) ProtoMessage() {}

func (x *GetBuildConfigsRequest) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[10]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBuildConfigsRequest.ProtoReflect.Descriptor instead.
func (*GetBuildConfigsRequest) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{10}
}

type GetBuildConfigsResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BuildConfigs  []*BuildConfig         `protobuf:"bytes,1,rep,name=build_configs,json=buildConfigs,proto3" json:"build_configs,omitempty"`
	CanGitPull    bool                   `protobuf:"varint,2,opt,name=can_git_pull,json=canGitPull,proto3" json:"can_git_pull,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetBuildConfigsResponse) Reset() {
	*x = GetBuildConfigsResponse{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[11]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetBuildConfigsResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBuildConfigsResponse) ProtoMessage() {}

func (x *GetBuildConfigsResponse) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[11]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBuildConfigsResponse.ProtoReflect.Descriptor instead.
func (*GetBuildConfigsResponse) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{11}
}

func (x *GetBuildConfigsResponse) GetBuildConfigs() []*BuildConfig {
	if x != nil {
		return x.BuildConfigs
	}
	return nil
}

func (x *GetBuildConfigsResponse) GetCanGitPull() bool {
	if x != nil {
		return x.CanGitPull
	}
	return false
}

type GetBuildConfigRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	ConfigName    string                 `protobuf:"bytes,1,opt,name=config_name,json=configName,proto3" json:"config_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetBuildConfigRequest) Reset() {
	*x = GetBuildConfigRequest{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[12]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetBuildConfigRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBuildConfigRequest) ProtoMessage() {}

func (x *GetBuildConfigRequest) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[12]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBuildConfigRequest.ProtoReflect.Descriptor instead.
func (*GetBuildConfigRequest) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{12}
}

func (x *GetBuildConfigRequest) GetConfigName() string {
	if x != nil {
		return x.ConfigName
	}
	return ""
}

type GetBuildConfigResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	BuildConfig   *BuildConfig           `protobuf:"bytes,1,opt,name=build_config,json=buildConfig,proto3" json:"build_config,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetBuildConfigResponse) Reset() {
	*x = GetBuildConfigResponse{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[13]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetBuildConfigResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetBuildConfigResponse) ProtoMessage() {}

func (x *GetBuildConfigResponse) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[13]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetBuildConfigResponse.ProtoReflect.Descriptor instead.
func (*GetBuildConfigResponse) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{13}
}

func (x *GetBuildConfigResponse) GetBuildConfig() *BuildConfig {
	if x != nil {
		return x.BuildConfig
	}
	return nil
}

type GetTemplateRequest struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	TemplateName  string                 `protobuf:"bytes,1,opt,name=template_name,json=templateName,proto3" json:"template_name,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetTemplateRequest) Reset() {
	*x = GetTemplateRequest{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[14]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetTemplateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetTemplateRequest) ProtoMessage() {}

func (x *GetTemplateRequest) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[14]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetTemplateRequest.ProtoReflect.Descriptor instead.
func (*GetTemplateRequest) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{14}
}

func (x *GetTemplateRequest) GetTemplateName() string {
	if x != nil {
		return x.TemplateName
	}
	return ""
}

type GetTemplateResponse struct {
	state         protoimpl.MessageState `protogen:"open.v1"`
	Template      *Template              `protobuf:"bytes,1,opt,name=template,proto3" json:"template,omitempty"`
	unknownFields protoimpl.UnknownFields
	sizeCache     protoimpl.SizeCache
}

func (x *GetTemplateResponse) Reset() {
	*x = GetTemplateResponse{}
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[15]
	ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
	ms.StoreMessageInfo(mi)
}

func (x *GetTemplateResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GetTemplateResponse) ProtoMessage() {}

func (x *GetTemplateResponse) ProtoReflect() protoreflect.Message {
	mi := &file_StencilBox_clientapi_v1_clientapi_proto_msgTypes[15]
	if x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GetTemplateResponse.ProtoReflect.Descriptor instead.
func (*GetTemplateResponse) Descriptor() ([]byte, []int) {
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP(), []int{15}
}

func (x *GetTemplateResponse) GetTemplate() *Template {
	if x != nil {
		return x.Template
	}
	return nil
}

var File_StencilBox_clientapi_v1_clientapi_proto protoreflect.FileDescriptor

const file_StencilBox_clientapi_v1_clientapi_proto_rawDesc = "" +
	"\n" +
	"'StencilBox/clientapi/v1/clientapi.proto\x12\x17StencilBox.clientapi.v1\"\r\n" +
	"\vInitRequest\"(\n" +
	"\fInitResponse\x12\x18\n" +
	"\aversion\x18\x01 \x01(\tR\aversion\"\xa0\x01\n" +
	"\bTemplate\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12\x16\n" +
	"\x06source\x18\x02 \x01(\tR\x06source\x12\x16\n" +
	"\x06status\x18\x03 \x01(\tR\x06status\x12+\n" +
	"\x11documentation_url\x18\x04 \x01(\tR\x10documentationUrl\x12#\n" +
	"\rbuild_configs\x18\x05 \x03(\tR\fbuildConfigs\"\xd8\x02\n" +
	"\vBuildConfig\x12\x12\n" +
	"\x04name\x18\x01 \x01(\tR\x04name\x12\x1a\n" +
	"\btemplate\x18\x02 \x01(\tR\btemplate\x12\x1d\n" +
	"\n" +
	"output_dir\x18\x03 \x01(\tR\toutputDir\x12\x14\n" +
	"\x05repos\x18\x04 \x03(\tR\x05repos\x12Q\n" +
	"\tdatafiles\x18\x05 \x03(\v23.StencilBox.clientapi.v1.BuildConfig.DatafilesEntryR\tdatafiles\x12\x1a\n" +
	"\bfilename\x18\x06 \x01(\tR\bfilename\x12\x12\n" +
	"\x04path\x18\a \x01(\tR\x04path\x12#\n" +
	"\rerror_message\x18\b \x01(\tR\ferrorMessage\x1a<\n" +
	"\x0eDatafilesEntry\x12\x10\n" +
	"\x03key\x18\x01 \x01(\tR\x03key\x12\x14\n" +
	"\x05value\x18\x02 \x01(\tR\x05value:\x028\x01\"/\n" +
	"\fBuildRequest\x12\x1f\n" +
	"\vconfig_name\x18\x01 \x01(\tR\n" +
	"configName\"\x9d\x01\n" +
	"\rBuildResponse\x12\x1f\n" +
	"\vconfig_name\x18\x01 \x01(\tR\n" +
	"configName\x12\x16\n" +
	"\x06status\x18\x02 \x01(\tR\x06status\x12\x14\n" +
	"\x05found\x18\x03 \x01(\bR\x05found\x12#\n" +
	"\rrelative_path\x18\x04 \x01(\tR\frelativePath\x12\x18\n" +
	"\aisError\x18\x05 \x01(\bR\aisError\"\x15\n" +
	"\x13GetTemplatesRequest\"W\n" +
	"\x14GetTemplatesResponse\x12?\n" +
	"\ttemplates\x18\x01 \x03(\v2!.StencilBox.clientapi.v1.TemplateR\ttemplates\"\x12\n" +
	"\x10GetStatusRequest\"\xaa\x01\n" +
	"\x11GetStatusResponse\x12!\n" +
	"\fin_container\x18\x02 \x01(\bR\vinContainer\x12%\n" +
	"\x0etemplates_path\x18\x04 \x01(\tR\rtemplatesPath\x12\x1f\n" +
	"\voutput_path\x18\x05 \x01(\tR\n" +
	"outputPath\x12*\n" +
	"\x11build_configs_dir\x18\x06 \x01(\tR\x0fbuildConfigsDir\"\x18\n" +
	"\x16GetBuildConfigsRequest\"\x86\x01\n" +
	"\x17GetBuildConfigsResponse\x12I\n" +
	"\rbuild_configs\x18\x01 \x03(\v2$.StencilBox.clientapi.v1.BuildConfigR\fbuildConfigs\x12 \n" +
	"\fcan_git_pull\x18\x02 \x01(\bR\n" +
	"canGitPull\"8\n" +
	"\x15GetBuildConfigRequest\x12\x1f\n" +
	"\vconfig_name\x18\x01 \x01(\tR\n" +
	"configName\"a\n" +
	"\x16GetBuildConfigResponse\x12G\n" +
	"\fbuild_config\x18\x01 \x01(\v2$.StencilBox.clientapi.v1.BuildConfigR\vbuildConfig\"9\n" +
	"\x12GetTemplateRequest\x12#\n" +
	"\rtemplate_name\x18\x01 \x01(\tR\ftemplateName\"T\n" +
	"\x13GetTemplateResponse\x12=\n" +
	"\btemplate\x18\x01 \x01(\v2!.StencilBox.clientapi.v1.TemplateR\btemplate2\xec\x05\n" +
	"\x14StencilBoxApiService\x12S\n" +
	"\x04Init\x12$.StencilBox.clientapi.v1.InitRequest\x1a%.StencilBox.clientapi.v1.InitResponse\x12[\n" +
	"\n" +
	"StartBuild\x12%.StencilBox.clientapi.v1.BuildRequest\x1a&.StencilBox.clientapi.v1.BuildResponse\x12k\n" +
	"\fGetTemplates\x12,.StencilBox.clientapi.v1.GetTemplatesRequest\x1a-.StencilBox.clientapi.v1.GetTemplatesResponse\x12h\n" +
	"\vGetTemplate\x12+.StencilBox.clientapi.v1.GetTemplateRequest\x1a,.StencilBox.clientapi.v1.GetTemplateResponse\x12b\n" +
	"\tGetStatus\x12).StencilBox.clientapi.v1.GetStatusRequest\x1a*.StencilBox.clientapi.v1.GetStatusResponse\x12t\n" +
	"\x0fGetBuildConfigs\x12/.StencilBox.clientapi.v1.GetBuildConfigsRequest\x1a0.StencilBox.clientapi.v1.GetBuildConfigsResponse\x12q\n" +
	"\x0eGetBuildConfig\x12..StencilBox.clientapi.v1.GetBuildConfigRequest\x1a/.StencilBox.clientapi.v1.GetBuildConfigResponseBJZHgithub.com/jamesread/StencilBox/gen/StencilBox/clientapi/v1;clientapi_pbb\x06proto3"

var (
	file_StencilBox_clientapi_v1_clientapi_proto_rawDescOnce sync.Once
	file_StencilBox_clientapi_v1_clientapi_proto_rawDescData []byte
)

func file_StencilBox_clientapi_v1_clientapi_proto_rawDescGZIP() []byte {
	file_StencilBox_clientapi_v1_clientapi_proto_rawDescOnce.Do(func() {
		file_StencilBox_clientapi_v1_clientapi_proto_rawDescData = protoimpl.X.CompressGZIP(unsafe.Slice(unsafe.StringData(file_StencilBox_clientapi_v1_clientapi_proto_rawDesc), len(file_StencilBox_clientapi_v1_clientapi_proto_rawDesc)))
	})
	return file_StencilBox_clientapi_v1_clientapi_proto_rawDescData
}

var file_StencilBox_clientapi_v1_clientapi_proto_msgTypes = make([]protoimpl.MessageInfo, 17)
var file_StencilBox_clientapi_v1_clientapi_proto_goTypes = []any{
	(*InitRequest)(nil),             // 0: StencilBox.clientapi.v1.InitRequest
	(*InitResponse)(nil),            // 1: StencilBox.clientapi.v1.InitResponse
	(*Template)(nil),                // 2: StencilBox.clientapi.v1.Template
	(*BuildConfig)(nil),             // 3: StencilBox.clientapi.v1.BuildConfig
	(*BuildRequest)(nil),            // 4: StencilBox.clientapi.v1.BuildRequest
	(*BuildResponse)(nil),           // 5: StencilBox.clientapi.v1.BuildResponse
	(*GetTemplatesRequest)(nil),     // 6: StencilBox.clientapi.v1.GetTemplatesRequest
	(*GetTemplatesResponse)(nil),    // 7: StencilBox.clientapi.v1.GetTemplatesResponse
	(*GetStatusRequest)(nil),        // 8: StencilBox.clientapi.v1.GetStatusRequest
	(*GetStatusResponse)(nil),       // 9: StencilBox.clientapi.v1.GetStatusResponse
	(*GetBuildConfigsRequest)(nil),  // 10: StencilBox.clientapi.v1.GetBuildConfigsRequest
	(*GetBuildConfigsResponse)(nil), // 11: StencilBox.clientapi.v1.GetBuildConfigsResponse
	(*GetBuildConfigRequest)(nil),   // 12: StencilBox.clientapi.v1.GetBuildConfigRequest
	(*GetBuildConfigResponse)(nil),  // 13: StencilBox.clientapi.v1.GetBuildConfigResponse
	(*GetTemplateRequest)(nil),      // 14: StencilBox.clientapi.v1.GetTemplateRequest
	(*GetTemplateResponse)(nil),     // 15: StencilBox.clientapi.v1.GetTemplateResponse
	nil,                             // 16: StencilBox.clientapi.v1.BuildConfig.DatafilesEntry
}
var file_StencilBox_clientapi_v1_clientapi_proto_depIdxs = []int32{
	16, // 0: StencilBox.clientapi.v1.BuildConfig.datafiles:type_name -> StencilBox.clientapi.v1.BuildConfig.DatafilesEntry
	2,  // 1: StencilBox.clientapi.v1.GetTemplatesResponse.templates:type_name -> StencilBox.clientapi.v1.Template
	3,  // 2: StencilBox.clientapi.v1.GetBuildConfigsResponse.build_configs:type_name -> StencilBox.clientapi.v1.BuildConfig
	3,  // 3: StencilBox.clientapi.v1.GetBuildConfigResponse.build_config:type_name -> StencilBox.clientapi.v1.BuildConfig
	2,  // 4: StencilBox.clientapi.v1.GetTemplateResponse.template:type_name -> StencilBox.clientapi.v1.Template
	0,  // 5: StencilBox.clientapi.v1.StencilBoxApiService.Init:input_type -> StencilBox.clientapi.v1.InitRequest
	4,  // 6: StencilBox.clientapi.v1.StencilBoxApiService.StartBuild:input_type -> StencilBox.clientapi.v1.BuildRequest
	6,  // 7: StencilBox.clientapi.v1.StencilBoxApiService.GetTemplates:input_type -> StencilBox.clientapi.v1.GetTemplatesRequest
	14, // 8: StencilBox.clientapi.v1.StencilBoxApiService.GetTemplate:input_type -> StencilBox.clientapi.v1.GetTemplateRequest
	8,  // 9: StencilBox.clientapi.v1.StencilBoxApiService.GetStatus:input_type -> StencilBox.clientapi.v1.GetStatusRequest
	10, // 10: StencilBox.clientapi.v1.StencilBoxApiService.GetBuildConfigs:input_type -> StencilBox.clientapi.v1.GetBuildConfigsRequest
	12, // 11: StencilBox.clientapi.v1.StencilBoxApiService.GetBuildConfig:input_type -> StencilBox.clientapi.v1.GetBuildConfigRequest
	1,  // 12: StencilBox.clientapi.v1.StencilBoxApiService.Init:output_type -> StencilBox.clientapi.v1.InitResponse
	5,  // 13: StencilBox.clientapi.v1.StencilBoxApiService.StartBuild:output_type -> StencilBox.clientapi.v1.BuildResponse
	7,  // 14: StencilBox.clientapi.v1.StencilBoxApiService.GetTemplates:output_type -> StencilBox.clientapi.v1.GetTemplatesResponse
	15, // 15: StencilBox.clientapi.v1.StencilBoxApiService.GetTemplate:output_type -> StencilBox.clientapi.v1.GetTemplateResponse
	9,  // 16: StencilBox.clientapi.v1.StencilBoxApiService.GetStatus:output_type -> StencilBox.clientapi.v1.GetStatusResponse
	11, // 17: StencilBox.clientapi.v1.StencilBoxApiService.GetBuildConfigs:output_type -> StencilBox.clientapi.v1.GetBuildConfigsResponse
	13, // 18: StencilBox.clientapi.v1.StencilBoxApiService.GetBuildConfig:output_type -> StencilBox.clientapi.v1.GetBuildConfigResponse
	12, // [12:19] is the sub-list for method output_type
	5,  // [5:12] is the sub-list for method input_type
	5,  // [5:5] is the sub-list for extension type_name
	5,  // [5:5] is the sub-list for extension extendee
	0,  // [0:5] is the sub-list for field type_name
}

func init() { file_StencilBox_clientapi_v1_clientapi_proto_init() }
func file_StencilBox_clientapi_v1_clientapi_proto_init() {
	if File_StencilBox_clientapi_v1_clientapi_proto != nil {
		return
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_StencilBox_clientapi_v1_clientapi_proto_rawDesc), len(file_StencilBox_clientapi_v1_clientapi_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   17,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_StencilBox_clientapi_v1_clientapi_proto_goTypes,
		DependencyIndexes: file_StencilBox_clientapi_v1_clientapi_proto_depIdxs,
		MessageInfos:      file_StencilBox_clientapi_v1_clientapi_proto_msgTypes,
	}.Build()
	File_StencilBox_clientapi_v1_clientapi_proto = out.File
	file_StencilBox_clientapi_v1_clientapi_proto_goTypes = nil
	file_StencilBox_clientapi_v1_clientapi_proto_depIdxs = nil
}
