package provider

import (
	webclient "axual-webclient"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &applicationDeploymentStateResource{}
	_ resource.ResourceWithImportState = &applicationDeploymentStateResource{}
)

// The desired states of an axual_application_deployment_state, and the observed state Read reports
// for a deployment whose connector, KSML app or Flink job has failed.
const (
	deploymentStateRunning = "RUNNING"
	deploymentStateStopped = "STOPPED"
	deploymentStateFailed  = "FAILED"
)

// runningWaitAttempts and runningWaitDelay bound how long a START or RESTART waits for the
// deployment to report itself running. Variables so the unit tests can shorten them.
var (
	runningWaitAttempts = 45
	runningWaitDelay    = 2 * time.Second
)

// resetWaitAttempts and resetWaitDelay bound how long a destroy waits, after STOP, for the API to
// offer RESET. RESET is only offered once the connector reports `Stopped`.
var (
	resetWaitAttempts = 15
	resetWaitDelay    = 2 * time.Second
)

func NewApplicationDeploymentStateResource(provider AxualProvider) resource.Resource {
	return &applicationDeploymentStateResource{provider: provider}
}

type applicationDeploymentStateResource struct {
	provider AxualProvider
}

type applicationDeploymentStateResourceData struct {
	Id                    types.String `tfsdk:"id"`
	ApplicationDeployment types.String `tfsdk:"application_deployment"`
	State                 types.String `tfsdk:"state"`
	CurrentState          types.String `tfsdk:"current_state"`
}

func (r *applicationDeploymentStateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_deployment_state"
}

func (r *applicationDeploymentStateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The desired running state of an Application Deployment: whether it should run or be stopped. " +
			"Use it with an `axual_application_deployment` that has `autostart = false`, so the deployment (its target and configs) can be created before the principal, credential and access grant, the same order the Self-Service UI uses. " +
			"Make this resource depend on the access grant approval and on the principal or credential. " +
			"Create and update send START or STOP and wait for the deployment to get there. " +
			"Every plan reads the real state, so a deployment that was stopped outside Terraform, or that has failed, shows up as a change and the next apply starts it again. " +
			"Destroy stops the deployment and, for a Connector, resets it (removes the connector from its Connect cluster, keeping the deployment, its configs and its offsets), so the principal, credential and grant can be removed after it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The Uid of the Application Deployment, the same as `application_deployment`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_deployment": schema.StringAttribute{
				MarkdownDescription: "The Uid of the `axual_application_deployment` whose state this resource manages.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The desired state: `RUNNING` or `STOPPED`. After a refresh this reads the state the deployment is really in, which is `FAILED` for a deployment that has failed, so drift shows up as a change back to the desired state.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(deploymentStateRunning, deploymentStateStopped),
				},
			},
			"current_state": schema.StringAttribute{
				MarkdownDescription: "The live status Platform Manager reports for the deployment, for example `Running`, `Starting`, `Stopped`, `Failed` or `Undefined` for a Connector.",
				Computed:            true,
			},
		},
	}
}

func (r *applicationDeploymentStateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data applicationDeploymentStateResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Id = data.ApplicationDeployment
	r.apply(ctx, &data, resp.Diagnostics.AddError)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *applicationDeploymentStateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data applicationDeploymentStateResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deploymentType, err := r.deploymentType(data.ApplicationDeployment.ValueString())
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			tflog.Info(ctx, fmt.Sprintf("Application Deployment %s no longer exists, removing its state from Terraform", data.ApplicationDeployment.ValueString()))
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read Application Deployment %s, got error: %s", data.ApplicationDeployment.ValueString(), err))
		return
	}
	status, err := r.provider.client.GetApplicationDeploymentStatus(data.ApplicationDeployment.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to read the status of Application Deployment %s, got error: %s", data.ApplicationDeployment.ValueString(), err))
		return
	}

	data.Id = data.ApplicationDeployment
	data.CurrentState = types.StringValue(liveDeploymentStatus(deploymentType, status))
	if observed := observedDeploymentState(deploymentType, status); observed != "" {
		data.State = types.StringValue(observed)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *applicationDeploymentStateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data applicationDeploymentStateResourceData
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Id = data.ApplicationDeployment
	r.apply(ctx, &data, resp.Diagnostics.AddError)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *applicationDeploymentStateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data applicationDeploymentStateResourceData
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	warning, err := r.resetAfterStop(ctx, data.ApplicationDeployment.ValueString())
	if err != nil && deploymentGone(r.provider.client, data.ApplicationDeployment.ValueString()) {
		// The deployment was deleted at the same time, for example by its own resource in this destroy.
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if warning != "" {
		resp.Diagnostics.AddWarning("Application Deployment was not reset", warning)
	}
}

// resetAfterStop stops the deployment and, for a Connector, resets it. A deployment that no longer
// exists has nothing left to stop.
//
// Platform Manager deletes an active Connector principal only once the deployment is back to how
// it was created (desired state UNDEPLOYED); a STOP leaves it at STOPPED. RESET is what takes it
// there: it removes the connector from its Connect cluster and keeps the deployment, its configs
// and its offsets. A KSML or Flink deployment needs nothing more than the STOP.
func (r *applicationDeploymentStateResource) resetAfterStop(ctx context.Context, id string) (string, error) {
	deploymentType, err := r.deploymentType(id)
	if err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			return "", nil
		}
		return "", fmt.Errorf("unable to read Application Deployment %s, got error: %s", id, err)
	}

	if err := stopDeploymentAndWait(ctx, r.provider.client, id, deploymentType); err != nil {
		return "", err
	}
	if isConnector(deploymentType) {
		return resetConnector(ctx, r.provider.client, id)
	}
	return "", nil
}

func (r *applicationDeploymentStateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := r.deploymentType(req.ID); err != nil {
		if errors.Is(err, webclient.NotFoundError) {
			resp.Diagnostics.AddError("Application Deployment Not Found", fmt.Sprintf("Application Deployment with ID: %s not found.", req.ID))
			return
		}
		resp.Diagnostics.AddError("API Error", fmt.Sprintf("Unable to find Application Deployment, got error: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("application_deployment"), req.ID)...)
}

// apply brings the deployment to the desired state in data and records the live status it ends in.
func (r *applicationDeploymentStateResource) apply(ctx context.Context, data *applicationDeploymentStateResourceData, addError func(string, string)) {
	id := data.ApplicationDeployment.ValueString()
	deploymentType, err := r.deploymentType(id)
	if err != nil {
		addError("API Error", fmt.Sprintf("Unable to read Application Deployment %s, got error: %s", id, err))
		return
	}

	switch data.State.ValueString() {
	case deploymentStateRunning:
		err = r.run(ctx, id, deploymentType)
	case deploymentStateStopped:
		err = r.stop(ctx, id, deploymentType)
	}
	if err != nil {
		addError("Application Deployment state not reached", err.Error())
		return
	}

	status, err := r.provider.client.GetApplicationDeploymentStatus(id)
	if err != nil {
		addError("API Error", fmt.Sprintf("Unable to read the status of Application Deployment %s, got error: %s", id, err))
		return
	}
	data.CurrentState = types.StringValue(liveDeploymentStatus(deploymentType, status))
}

// run starts the deployment, or restarts it when it has failed, and waits until it runs.
func (r *applicationDeploymentStateResource) run(ctx context.Context, id string, deploymentType string) error {
	status, err := r.provider.client.GetApplicationDeploymentStatus(id)
	if err != nil {
		return fmt.Errorf("unable to get Application Deployment status: %s", err)
	}
	switch observedDeploymentState(deploymentType, status) {
	case deploymentStateRunning:
		if isDeploymentRunning(deploymentType, status) {
			tflog.Info(ctx, fmt.Sprintf("Application Deployment %s is already running", id))
			return nil
		}
		// Starting: nothing to send, only wait.
	case deploymentStateFailed:
		// A failed connector is still on its Connect cluster: the API offers RESTART, not START.
		restarted, err := restartFailedConnector(ctx, r.provider.client, id, status)
		if err != nil {
			return err
		}
		if restarted {
			return r.waitUntilRunning(ctx, id, deploymentType, failedPollsAfterRestart)
		}
		fallthrough
	default:
		if err := startApplicationDeployment(ctx, r.provider.client, id, deploymentType, 5*time.Second); err != nil {
			return err
		}
	}
	return r.waitUntilRunning(ctx, id, deploymentType, failedPollsAfterRestart)
}

// failedPollsAfterRestart is how many status reads may still show a failure after a start or restart:
// Connect restarts tasks in the background, and a resumed task can fail once on credentials it read
// before they changed, then restart with the new ones.
const failedPollsAfterRestart = 5

// restartFailedConnector restarts a failed connector and, one by one, its failed tasks: a connector
// RESTART leaves failed tasks failed. It reports whether it restarted anything.
func restartFailedConnector(ctx context.Context, client *webclient.Client, id string, status *webclient.ApplicationDeploymentStatusResponse) (bool, error) {
	restarted := false
	if strings.EqualFold(status.ConnectorState.State, "Failed") && status.Links.Has(webclient.RelRestart) {
		tflog.Info(ctx, fmt.Sprintf("Application Deployment %s has failed, restarting it", id))
		if err := operateDeployment(client, id, actionRestart); err != nil {
			return false, fmt.Errorf("unable to restart the failed Application Deployment: %s", err)
		}
		restarted = true
	}
	for _, task := range status.TaskStates {
		if !strings.EqualFold(task.Status, "Failed") {
			continue
		}
		tflog.Info(ctx, fmt.Sprintf("Task %d of Application Deployment %s has failed, restarting it", task.Id, id))
		if err := client.RestartApplicationDeploymentTask(id, task.Id); err != nil {
			return false, fmt.Errorf("unable to restart failed task %d: %s", task.Id, err)
		}
		restarted = true
	}
	return restarted, nil
}

// stop stops the deployment and fails when it is still running afterwards.
func (r *applicationDeploymentStateResource) stop(ctx context.Context, id string, deploymentType string) error {
	if err := stopDeploymentAndWait(ctx, r.provider.client, id, deploymentType); err != nil {
		return err
	}
	status, err := r.provider.client.GetApplicationDeploymentStatus(id)
	if err != nil {
		return fmt.Errorf("unable to get Application Deployment status after stopping it: %s", err)
	}
	if isDeploymentRunning(deploymentType, status) {
		return fmt.Errorf("the deployment is still running after STOP (%s)", describeDeploymentStatus(status))
	}
	return nil
}

func (r *applicationDeploymentStateResource) waitUntilRunning(ctx context.Context, id string, deploymentType string, failedPollsAllowed int) error {
	var status *webclient.ApplicationDeploymentStatusResponse
	var err error
	for i := 0; i < runningWaitAttempts; i++ {
		status, err = r.provider.client.GetApplicationDeploymentStatus(id)
		if err != nil {
			return fmt.Errorf("unable to get Application Deployment status while waiting for it to run: %s", err)
		}
		switch observedDeploymentState(deploymentType, status) {
		case deploymentStateFailed:
			if i >= failedPollsAllowed {
				return fmt.Errorf("the deployment failed after it was started (%s)%s", describeDeploymentStatus(status), failureTrace(status))
			}
		case deploymentStateRunning:
			if isDeploymentRunning(deploymentType, status) {
				tflog.Info(ctx, fmt.Sprintf("Application Deployment %s is running", id))
				return nil
			}
		}
		time.Sleep(runningWaitDelay)
	}
	return fmt.Errorf("the deployment did not report itself running within %s (%s)", time.Duration(runningWaitAttempts)*runningWaitDelay, describeDeploymentStatus(status))
}

// connectorStateFailed is the live state Platform Manager reports for a failed connector.
const connectorStateFailed = "Failed"

// resetConnector sends RESET once the API offers it. When it is never offered, it returns a warning
// rather than an error: the destroy can still go on, but deleting the active principal may then fail.
func resetConnector(ctx context.Context, client *webclient.Client, id string) (string, error) {
	state := ""
	for i := 0; i < resetWaitAttempts; i++ {
		status, err := client.GetApplicationDeploymentStatus(id)
		if err != nil {
			return "", fmt.Errorf("unable to get Application Deployment status before resetting it: %s", err)
		}
		state = status.ConnectorState.State
		if status.Links.Has(webclient.RelReset) {
			return "", sendReset(ctx, client, id)
		}
		if state == connectorStateFailed {
			// A failed connector stays Failed after STOP, so waiting for RESET does not help.
			break
		}
		if !status.Links.Has(webclient.RelStop) && status.Links.Has(webclient.RelStart) && status.ConnectorState.State != "Stopped" {
			// Nothing is deployed on the Connect cluster any more (for example never started, or already reset).
			return "", nil
		}
		time.Sleep(resetWaitDelay)
	}
	if state == connectorStateFailed {
		return fmt.Sprintf("Application Deployment %s could not be reset: its connector is Failed, and this Platform Manager version "+
			"does not allow RESET for a failed connector. Deleting its active principal can fail. Fix the cause and restart the "+
			"connector, or delete the deployment in the Self-Service UI, then apply again.", id), nil
	}
	if state == "" {
		state = "unknown"
	}
	return fmt.Sprintf("Application Deployment %s was stopped, but Platform Manager did not offer RESET within %s. "+
		"Its connector is %s, so deleting its active principal can fail. Reset it in the Self-Service UI, then apply again.",
		id, time.Duration(resetWaitAttempts)*resetWaitDelay, state), nil
}

func sendReset(ctx context.Context, client *webclient.Client, id string) error {
	if err := operateDeployment(client, id, actionReset); err != nil {
		return fmt.Errorf("unable to reset Application Deployment %s: %s", id, err)
	}
	tflog.Info(ctx, fmt.Sprintf("Reset Application Deployment %s", id))
	return nil
}

func (r *applicationDeploymentStateResource) deploymentType(id string) (string, error) {
	deployment, err := r.provider.client.GetApplicationDeployment(id)
	if err != nil {
		return "", err
	}
	if deployment.Embedded.Application.ApplicationType == "" {
		return connectorApplicationType, nil
	}
	return deployment.Embedded.Application.ApplicationType, nil
}

// observedDeploymentState maps the live status to RUNNING, STOPPED or FAILED, or "" when the
// status carries nothing to go by. Starting counts as RUNNING: the deployment is on its way there.
// Connect keeps a connector `Running` while one of its tasks has failed, so a failed task counts as
// FAILED: that connector moves no data.
func observedDeploymentState(deploymentType string, status *webclient.ApplicationDeploymentStatusResponse) string {
	if isConnector(deploymentType) && status.ConnectorState.State == "Running" {
		for _, task := range status.TaskStates {
			if strings.EqualFold(task.Status, "Failed") {
				return deploymentStateFailed
			}
		}
	}
	return mapLiveStatus(liveDeploymentStatus(deploymentType, status))
}

func mapLiveStatus(live string) string {
	switch strings.ToLower(live) {
	case "":
		return ""
	case "running", "starting":
		return deploymentStateRunning
	case "failed", "failing":
		return deploymentStateFailed
	default:
		return deploymentStateStopped
	}
}

// failureTrace is the first line of the connector's or a failed task's trace, if the API gave one.
func failureTrace(status *webclient.ApplicationDeploymentStatusResponse) string {
	trace := status.ConnectorState.Trace
	for _, task := range status.TaskStates {
		if trace == "" && task.Trace != "" {
			trace = task.Trace
		}
	}
	if trace == "" {
		return ""
	}
	return ": " + strings.SplitN(trace, "\n", 2)[0]
}
