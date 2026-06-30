<template>
  <ResourcePublishForm :initial-options="initialOptions" mode="create" :reserve-bottom-safe-area="false" />
</template>

<script setup>
import ResourcePublishForm from '../../components/ResourcePublishForm.vue'

const initialOptions = {}
</script>
