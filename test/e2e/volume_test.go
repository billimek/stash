package e2e_test

import (
	"github.com/appscode/go/crypto/rand"
	"github.com/appscode/stash/apis"
	"github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	core "k8s.io/api/core/v1"
	v1 "kmodules.xyz/client-go/core/v1"
)

var (
	bpv              *core.PersistentVolume
	bpvc             *core.PersistentVolumeClaim
	rpvc             *core.PersistentVolumeClaim
	pod              core.Pod
	updateStatusFunc v1beta1.Function
	backupFunc       v1beta1.Function
	restoreFunc      v1beta1.Function
	backupTask       v1beta1.Task
	restoreTask      v1beta1.Task
)

var _ = Describe("Volume", func() {
	BeforeEach(func() {
		f = root.Invoke()

		By("Creating functions")
		updateStatusFunc = f.UpdateStatusFunction()
		backupFunc = f.PvcBackupFunction()
		restoreFunc = f.PvcRestoreFunction()

		err = f.CreateFunction(updateStatusFunc)
		Expect(err).NotTo(HaveOccurred())
		err = f.CreateFunction(backupFunc)
		Expect(err).NotTo(HaveOccurred())
		err = f.CreateFunction(restoreFunc)
		Expect(err).NotTo(HaveOccurred())

		By("Creating Tasks")
		backupTask = f.BackupTask()
		restoreTask = f.RestoreTask()

		err = f.CreateTask(backupTask)
		Expect(err).NotTo(HaveOccurred())
		err = f.CreateTask(restoreTask)
		Expect(err).NotTo(HaveOccurred())

	})
	JustBeforeEach(func() {
		pod = f.Pod(bpvc.Name)
		cred = f.SecretForLocalBackend()
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		pvc = f.GetPersistentVolumeClaim()
		err = f.CreatePersistentVolumeClaim(pvc)
		Expect(err).NotTo(HaveOccurred())

		repo = f.Repository(cred.Name, pvc.Name)

		backupCfg = f.BackupConfiguration(repo.Name, targetref)
		backupCfg.Spec.Target = f.PvcBackupTarget(bpvc.Name)
		backupCfg.Spec.Task.Name = backupTask.Name

		restoreSession = f.RestoreSession(repo.Name, targetref, rules)
		restoreSession.Spec.Target = f.PvcRestoreTarget(bpvc.Name)
		restoreSession.Spec.Rules = []v1beta1.Rule{
			{
				Paths: []string{
					framework.TestSourceDataMountPath,
				},
			},
		}
		restoreSession.Spec.Task.Name = restoreTask.Name

	})
	AfterEach(func() {
		err = f.DeleteFunction(updateStatusFunc.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = f.DeleteFunction(backupFunc.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = f.DeleteFunction(restoreFunc.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = f.DeleteTask(backupTask.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = f.DeleteTask(restoreTask.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())

		err = f.DeleteSecret(cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
		err = framework.WaitUntilSecretDeleted(f.KubeClient, cred.ObjectMeta)
		Expect(err).NotTo(HaveOccurred())
	})
	var (
		testPVCBackup = func() {
			By("Creating New PVC")
			err = f.CreatePersistentVolumeClaim(bpvc)
			Expect(err).NotTo(HaveOccurred())

			By("Create Pod and Generate sample Data")
			err = f.CreatePod(pod)
			Expect(err).NotTo(HaveOccurred())
			err = v1.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Creating sample data inside Pod")
			err = f.CreateSampleDataInsideWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data directory inside pod")
			sampleData, err = f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())

			By("Creating storage Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating new repository")
			err = f.CreateRepository(repo)
			Expect(err).NotTo(HaveOccurred())

			By("Creating BackupConfiguration" + backupCfg.Name)
			err = f.CreateBackupConfiguration(backupCfg)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for BackupSession")
			f.EventuallyBackupSessionCreated(backupCfg.ObjectMeta).Should(BeTrue())
			bs, err := f.GetBackupSession(backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Check for succeeded BackupSession")
			f.EventuallyBackupSessionPhase(bs.ObjectMeta).Should(Equal(v1beta1.BackupSessionSucceeded))

			By("Delete BackupConfiguration")
			err = f.DeleteBackupConfiguration(backupCfg)
			err = framework.WaitUntilBackupConfigurationDeleted(f.StashClient, backupCfg.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Remove sample data from PVC")
			err = f.CleanupSampleDataFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			err = v1.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		}
	)
	Context("General Backup && Restore for PVC Volume", func() {
		BeforeEach(func() {
			bpvc = f.GetPersistentVolumeClaim()
		})
		AfterEach(func() {
			err = f.DeletePersistentVolumeClaim(bpvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRestoreSessionDeleted(f.StashClient, restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("General Backup new PVC", func() {
			By("new backup for PVC")
			testPVCBackup()

			By("Creating Restore Session")
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Reading sample data from /source/data directory inside pod")
			restoredData, err = f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeletePod(pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying restored data is same as original data")
			Expect(sampleData).To(BeEquivalentTo(restoredData))

		})
	})

	Context("Restore data on different PVC", func() {
		BeforeEach(func() {
			bpvc = f.GetPersistentVolumeClaim()
			rpvc = f.GetPersistentVolumeClaim()
		})
		AfterEach(func() {
			err = f.DeletePersistentVolumeClaim(bpvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeletePersistentVolumeClaim(rpvc.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRepository(repo)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRepositoryDeleted(f.StashClient, repo)
			Expect(err).NotTo(HaveOccurred())

			err = f.DeleteRestoreSession(restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())
			err = framework.WaitUntilRestoreSessionDeleted(f.StashClient, restoreSession.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

		})
		It("General Backup new PVC", func() {
			By("new backup for PVC")
			testPVCBackup()

			By("Create another PVC")
			err := f.CreatePersistentVolumeClaim(rpvc)

			By("Creating Restore Session")
			restoreSession.Spec.Target.Ref.Name = rpvc.Name
			err = f.CreateRestoreSession(restoreSession)
			Expect(err).NotTo(HaveOccurred())

			By("Waiting for restore to succeed")
			f.EventuallyRestoreSessionPhase(restoreSession.ObjectMeta).Should(Equal(v1beta1.RestoreSessionSucceeded))

			By("Delete previous Pod")
			err = f.DeletePod(pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Create another Pod with bounded new PVC")
			pod.Name = rand.WithUniqSuffix("restore-test")
			pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName = rpvc.Name
			err = f.CreatePod(pod)
			Expect(err).NotTo(HaveOccurred())
			err = v1.WaitUntilPodRunning(f.KubeClient, pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Reading sample data from /source/data directory inside pod")
			restoredData, err = f.ReadSampleDataFromFromWorkload(pod.ObjectMeta, apis.KindPersistentVolumeClaim)
			Expect(err).NotTo(HaveOccurred())
			err = f.DeletePod(pod.ObjectMeta)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying restored data is same as original data")
			Expect(sampleData).To(BeEquivalentTo(restoredData))

		})
	})
})
